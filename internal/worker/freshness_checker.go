// Package worker contains background goroutines that run continuously inside
// the server process.
package worker

import (
	"context"
	"log/slog"
	"time"
)

// FreshnessDecayer is the subset of FreshnessService needed by the worker.
type FreshnessDecayer interface {
	RunDecay(ctx context.Context) error
}

// RunFreshnessChecker runs the freshness decay scoring job on a fixed interval.
// It returns when ctx is cancelled.
func RunFreshnessChecker(ctx context.Context, logger *slog.Logger, svc FreshnessDecayer, interval time.Duration) {
	if interval <= 0 {
		interval = 6 * time.Hour
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info("freshness checker started", slog.Duration("interval", interval))

	for {
		select {
		case <-ctx.Done():
			logger.Info("freshness checker stopped")
			return
		case t := <-ticker.C:
			logger.Info("freshness decay run starting", slog.Time("tick", t))
			if err := svc.RunDecay(ctx); err != nil {
				logger.Error("freshness decay run failed", slog.String("error", err.Error()))
			} else {
				logger.Info("freshness decay run complete")
			}
		}
	}
}
