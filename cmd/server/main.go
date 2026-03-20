// Package main is the entry point for the Knomantem API server.
// It performs full dependency injection, starts background workers, configures
// the Echo HTTP server, and handles graceful shutdown.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"

	"github.com/knomantem/knomantem/internal/config"
	"github.com/knomantem/knomantem/internal/handler"
	mw "github.com/knomantem/knomantem/internal/middleware"
	"github.com/knomantem/knomantem/internal/repository/postgres"
	"github.com/knomantem/knomantem/internal/service"
	"github.com/knomantem/knomantem/internal/worker"
	"github.com/knomantem/knomantem/pkg/search"
)

func main() {
	// ── Logger ────────────────────────────────────────────────────────────────
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if cfg.LogLevel == "debug" {
		logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	// ── Database ──────────────────────────────────────────────────────────────
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	db, err := postgres.NewDB(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// ── Repositories ──────────────────────────────────────────────────────────
	userRepo := postgres.NewUserRepo(db)
	spaceRepo := postgres.NewSpaceRepo(db)
	pageRepo := postgres.NewPageRepo(db)
	freshnessRepo := postgres.NewFreshnessRepo(db)
	edgeRepo := postgres.NewEdgeRepo(db)
	tagRepo := postgres.NewTagRepo(db)
	notificationRepo := postgres.NewNotificationRepo(db)

	// ── Search index ──────────────────────────────────────────────────────────
	bleveIndex, err := search.NewBleveIndex(cfg.BleveIndexPath)
	if err != nil {
		logger.Error("failed to open bleve index", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer bleveIndex.Close()

	// SearchRepoAdapter bridges pkg/search.BleveSearch to domain.SearchRepository.
	searchRepo := search.NewSearchRepoAdapter(bleveIndex)

	// ── Services ──────────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiry, cfg.RefreshExpiry)
	spaceSvc := service.NewSpaceService(spaceRepo)
	pageSvc := service.NewPageService(pageRepo, searchRepo, edgeRepo, tagRepo, freshnessRepo)
	searchSvc := service.NewSearchService(searchRepo, pageRepo, freshnessRepo)
	freshnessSvc := service.NewFreshnessService(freshnessRepo, pageRepo, notificationRepo)
	graphSvc := service.NewGraphService(edgeRepo, pageRepo)
	tagSvc := service.NewTagService(tagRepo)
	presenceSvc := service.NewPresenceService()

	// ── Background workers ────────────────────────────────────────────────────
	workerCtx, workerCancel := context.WithCancel(context.Background())

	go worker.RunFreshnessChecker(workerCtx, logger, freshnessSvc, cfg.FreshnessInterval)
	go worker.RunSearchIndexer(workerCtx, logger, pageRepo, searchRepo)

	logger.Info("background workers started")

	// ── Echo setup ────────────────────────────────────────────────────────────
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Global middleware
	e.Use(mw.CORS(cfg.CORSOrigins))
	e.Use(mw.Logger(logger))
	e.Use(echo.WrapMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Attach a request-id header for correlation
			if r.Header.Get("X-Request-ID") == "" {
				r.Header.Set("X-Request-ID", generateRequestID())
			}
			next.ServeHTTP(w, r)
		})
	}))
	e.Use(mw.RateLimiter(rate.Limit(cfg.RateLimitRPS), cfg.RateLimitBurst))

	// ── Handlers & Routes ─────────────────────────────────────────────────────
	presenceWS := handler.NewPresenceHandler(presenceSvc, parseOrigins(cfg.CORSOrigins))

	handlers := &handler.Handlers{
		Auth:            handler.NewAuthHandler(authSvc),
		Space:           handler.NewSpaceHandler(spaceSvc),
		Page:            handler.NewPageHandler(pageSvc),
		Search:          handler.NewSearchHandler(searchSvc),
		Freshness:       handler.NewFreshnessHandler(freshnessSvc),
		Graph:           handler.NewGraphHandler(graphSvc),
		Tag:             handler.NewTagHandler(tagSvc),
		Presence:        presenceWS,
		PresenceViewers: handler.NewPresenceHTTPHandler(presenceSvc),
	}

	handler.RegisterRoutes(e, handlers, cfg.JWTSecret)

	// ── Start server ──────────────────────────────────────────────────────────
	addr := ":" + cfg.Port
	logger.Info("starting HTTP server", slog.String("addr", addr))

	serverErr := make(chan error, 1)
	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		logger.Error("server error", slog.String("error", err.Error()))
	case sig := <-quit:
		logger.Info("shutdown signal received", slog.String("signal", sig.String()))
	}

	// Stop background workers first.
	workerCancel()

	// Give the HTTP server 10 seconds to drain.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
	} else {
		logger.Info("server shut down cleanly")
	}
}

// generateRequestID returns a simple request ID for correlation logging.
// In production you may want to use a proper UUID or ULID here.
func generateRequestID() string {
	return time.Now().Format("20060102150405.000000000")
}

// parseOrigins splits a comma-separated origins string into a slice.
func parseOrigins(s string) []string {
	if s == "" || s == "*" {
		return []string{"*"}
	}
	var out []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := trimSpace(s[start:i])
			if part != "" {
				out = append(out, part)
			}
			start = i + 1
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
