# Concurrency Reference — Goroutines and Background Workers

> Patterns for goroutine lifecycle, worker pools, error groups, channels, and leak prevention.
> Go 1.22+.

---

## 1. Background Worker Pattern

The canonical pattern for a background worker that:
- Runs until the application shuts down.
- Respects `context.Context` cancellation.
- Logs errors without crashing.
- Participates in graceful shutdown via `sync.WaitGroup` or `errgroup`.

```go
// internal/worker/index_worker.go

package worker

import (
    "context"
    "log/slog"
    "time"

    "yourapp/internal/domain"
)

type IndexWorker struct {
    repo   domain.DocumentRepository
    search domain.SearchIndex
    logger *slog.Logger
}

func NewIndexWorker(repo domain.DocumentRepository, search domain.SearchIndex, logger *slog.Logger) *IndexWorker {
    return &IndexWorker{repo: repo, search: search, logger: logger}
}

// Run blocks until ctx is cancelled. Call in a goroutine from main.
func (w *IndexWorker) Run(ctx context.Context) {
    w.logger.Info("index worker started")
    defer w.logger.Info("index worker stopped")

    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    // Run once immediately, then on each tick.
    w.runOnce(ctx)

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            w.runOnce(ctx)
        }
    }
}

func (w *IndexWorker) runOnce(ctx context.Context) {
    if err := w.processQueue(ctx); err != nil {
        // Don't crash the worker on transient errors.
        w.logger.ErrorContext(ctx, "index worker error", "error", err)
    }
}

func (w *IndexWorker) processQueue(ctx context.Context) error {
    // Check cancellation before doing expensive work.
    if ctx.Err() != nil {
        return ctx.Err()
    }
    // ... do work ...
    return nil
}
```

### Wiring in main.go

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // ... build dependencies ...

    var wg sync.WaitGroup

    wg.Add(1)
    go func() {
        defer wg.Done()
        indexWorker.Run(ctx)
    }()

    wg.Add(1)
    go func() {
        defer wg.Done()
        cleanupWorker.Run(ctx)
    }()

    // Start HTTP server — shut it down when ctx is cancelled.
    if err := runServer(ctx, e); err != nil {
        logger.Error("server error", "error", err)
    }

    // Wait for all workers to finish after context is cancelled.
    wg.Wait()
    logger.Info("all workers stopped, exiting")
}
```

---

## 2. errgroup vs sync.WaitGroup

### sync.WaitGroup — use when errors don't matter or are handled separately

```go
var wg sync.WaitGroup

for _, item := range items {
    item := item // capture loop variable (required before Go 1.22)
    wg.Add(1)
    go func() {
        defer wg.Done()
        process(item)
    }()
}

wg.Wait()
```

**Limitations:** Cannot propagate errors. If any goroutine fails, you need a separate channel to collect errors.

### errgroup — use when you need the first error and automatic cancellation

```go
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(ctx)

for _, item := range items {
    item := item
    g.Go(func() error {
        return process(ctx, item) // ctx is cancelled if any goroutine returns an error
    })
}

if err := g.Wait(); err != nil {
    return fmt.Errorf("processing failed: %w", err)
}
```

`errgroup.WithContext` returns a derived context that is cancelled when:
- Any function passed to `g.Go()` returns a non-nil error, OR
- `g.Wait()` returns.

This makes errgroup the correct choice for fan-out work where any single failure should cancel all remaining work.

### errgroup with Concurrency Limit

```go
g, ctx := errgroup.WithContext(ctx)
g.SetLimit(10) // at most 10 goroutines active at a time

for _, id := range ids {
    id := id
    g.Go(func() error {
        return fetchAndProcess(ctx, id)
    })
}

if err := g.Wait(); err != nil {
    return err
}
```

Use `g.TryGo()` when you want non-blocking behaviour — it returns `false` if the goroutine limit is reached rather than blocking.

---

## 3. Worker Pool Pattern

When you need a fixed-size pool of workers consuming from a queue:

```go
// internal/worker/pool.go

package worker

import (
    "context"
    "log/slog"
    "sync"
)

type Job func(ctx context.Context) error

type Pool struct {
    jobs    chan Job
    workers int
    logger  *slog.Logger
}

func NewPool(workers, queueSize int, logger *slog.Logger) *Pool {
    return &Pool{
        jobs:    make(chan Job, queueSize),
        workers: workers,
        logger:  logger,
    }
}

// Start launches worker goroutines. Call once from main.
func (p *Pool) Start(ctx context.Context, wg *sync.WaitGroup) {
    for i := range p.workers {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            p.runWorker(ctx, workerID)
        }(i)
    }
}

func (p *Pool) runWorker(ctx context.Context, id int) {
    for {
        select {
        case <-ctx.Done():
            return
        case job, ok := <-p.jobs:
            if !ok {
                return // channel closed
            }
            if err := job(ctx); err != nil {
                p.logger.ErrorContext(ctx, "worker job failed", "worker_id", id, "error", err)
            }
        }
    }
}

// Submit adds a job to the queue. Returns false if the queue is full.
func (p *Pool) Submit(job Job) bool {
    select {
    case p.jobs <- job:
        return true
    default:
        return false // drop or handle backpressure
    }
}

// Close signals workers to stop after draining the queue. Call after context is cancelled.
func (p *Pool) Close() {
    close(p.jobs)
}
```

---

## 4. Channel Patterns

### Buffered vs Unbuffered

| Use Case | Channel Type | Reason |
|---|---|---|
| Synchronisation (handoff) | Unbuffered `make(chan T)` | Sender blocks until receiver is ready — guarantees delivery |
| Work queue with known capacity | Buffered `make(chan T, n)` | Decouples producer from consumer; backpressure when full |
| Fire-and-forget signals | Buffered `make(chan struct{}, 1)` | Non-blocking signal; receiver drains at its own pace |
| Pipeline stages | Unbuffered | Back-pressure propagates naturally up the pipeline |
| Rate limiting (semaphore) | Buffered `make(chan struct{}, n)` | Limit concurrency to n slots |

### Semaphore Pattern (Limit Concurrency)

```go
sem := make(chan struct{}, 5) // allow 5 concurrent operations

for _, item := range items {
    sem <- struct{}{}       // acquire slot (blocks when full)
    go func(item Item) {
        defer func() { <-sem }() // release slot
        process(item)
    }(item)
}

// Wait for all slots to be released.
for i := 0; i < cap(sem); i++ {
    sem <- struct{}{}
}
```

Prefer `errgroup.SetLimit()` over manual semaphores for new code — it is cleaner and handles errors.

### Done Channel Pattern

```go
// Canonical pattern for goroutines that produce values until cancelled.
func generate(ctx context.Context) <-chan int {
    ch := make(chan int)
    go func() {
        defer close(ch) // always close the channel when done
        n := 0
        for {
            select {
            case <-ctx.Done():
                return
            case ch <- n:
                n++
            }
        }
    }()
    return ch
}
```

Always `close()` a channel from the **sender**, never the receiver. Closing signals "no more values". Receiving from a closed channel returns the zero value immediately with `ok == false`.

### Fan-Out / Fan-In

```go
// Fan-out: distribute work to N workers.
func fanOut(ctx context.Context, in <-chan Job, n int) []<-chan Result {
    outs := make([]<-chan Result, n)
    for i := range n {
        outs[i] = worker(ctx, in)
    }
    return outs
}

// Fan-in: merge N channels into one.
func fanIn(ctx context.Context, channels ...<-chan Result) <-chan Result {
    out := make(chan Result)
    var wg sync.WaitGroup
    for _, ch := range channels {
        ch := ch
        wg.Add(1)
        go func() {
            defer wg.Done()
            for v := range ch {
                select {
                case out <- v:
                case <-ctx.Done():
                    return
                }
            }
        }()
    }
    go func() {
        wg.Wait()
        close(out)
    }()
    return out
}
```

---

## 5. Avoiding Goroutine Leaks

A goroutine leak occurs when a goroutine is started but never terminates. Over time, leaked goroutines exhaust memory and file descriptors.

### Common Causes and Fixes

**Blocked send on an unbuffered channel with no receiver:**
```go
// BAD: if no one reads from ch, the goroutine leaks forever.
go func() {
    result := compute()
    ch <- result // blocks if caller has already returned
}()

// GOOD: use a buffered channel sized to the number of senders.
ch := make(chan Result, 1)
go func() {
    ch <- compute() // never blocks — buffer absorbs the value
}()
```

**Missing context check in a loop:**
```go
// BAD: loop continues even after context is cancelled.
func (w *Worker) Run() {
    for {
        w.process()
        time.Sleep(1 * time.Second)
    }
}

// GOOD: always select on ctx.Done().
func (w *Worker) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
        }
        w.process(ctx)
        select {
        case <-ctx.Done():
            return
        case <-time.After(1 * time.Second):
        }
    }
}
```

**HTTP response body not read and closed:**
```go
// BAD: goroutines inside http.Client will leak if body is not drained.
resp, _ := http.Get(url)
if resp.StatusCode != 200 { return }

// GOOD: always drain and close the body.
defer func() {
    io.Copy(io.Discard, resp.Body)
    resp.Body.Close()
}()
```

### Testing for Leaks

Use `goleak` in tests to assert that no goroutines are leaked:

```go
import "go.uber.org/goleak"

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}

// Or per-test:
func TestMyFunction(t *testing.T) {
    defer goleak.VerifyNone(t)
    // ... test code ...
}
```

---

## 6. Graceful Shutdown Sequence

```go
// cmd/server/main.go

func main() {
    // 1. Create a root context that cancels on OS signal.
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // 2. Start background workers — pass ctx so they stop on signal.
    var wg sync.WaitGroup
    for _, w := range workers {
        wg.Add(1)
        go func(worker Worker) {
            defer wg.Done()
            worker.Run(ctx)
        }(w)
    }

    // 3. Start HTTP server in a goroutine.
    serverErr := make(chan error, 1)
    go func() {
        if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
            serverErr <- err
        }
    }()

    // 4. Block until signal or server error.
    select {
    case <-ctx.Done():
        logger.Info("shutdown signal received")
    case err := <-serverErr:
        logger.Error("server error", "error", err)
        stop() // cancel ctx to stop workers too
    }

    // 5. Gracefully shutdown the HTTP server (stop accepting new requests).
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    if err := e.Shutdown(shutdownCtx); err != nil {
        logger.Error("server shutdown error", "error", err)
    }

    // 6. Wait for background workers to drain and finish.
    wg.Wait()
    logger.Info("clean shutdown complete")
}
```

The shutdown sequence is:
1. Signal received → root ctx cancelled.
2. HTTP server stops accepting new connections.
3. In-flight HTTP requests complete (up to shutdown timeout).
4. Background workers detect `ctx.Done()` and return.
5. `wg.Wait()` unblocks — process exits.

---

## 7. Recover in Goroutines

A panic in a goroutine that is not recovered will crash the entire process. Always add a recovery wrapper in long-running goroutines.

```go
func safeGo(ctx context.Context, logger *slog.Logger, name string, fn func(context.Context)) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                // log stack trace
                buf := make([]byte, 64<<10)
                n := runtime.Stack(buf, false)
                logger.ErrorContext(ctx, "goroutine panic recovered",
                    "goroutine", name,
                    "panic",     r,
                    "stack",     string(buf[:n]),
                )
            }
        }()
        fn(ctx)
    }()
}

// Usage:
safeGo(ctx, logger, "IndexWorker", indexWorker.Run)
```

Echo's `middleware.Recover()` handles panics in HTTP handlers — but **not** in goroutines launched from handlers. If a handler spawns a goroutine, wrap it with `safeGo`.
