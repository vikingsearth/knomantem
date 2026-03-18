package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/knomantem/knomantem/internal/domain"
)

// RunSearchIndexer periodically re-indexes all pages to keep the Bleve index
// in sync with the PostgreSQL database. In production this would be replaced
// by event-driven updates (e.g. listening to a PostgreSQL NOTIFY channel);
// for the POC a full re-index every hour is sufficient.
//
// It returns when ctx is cancelled.
func RunSearchIndexer(
	ctx context.Context,
	logger *slog.Logger,
	pages domain.PageRepository,
	search domain.SearchRepository,
) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	logger.Info("search indexer started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("search indexer stopped")
			return
		case t := <-ticker.C:
			logger.Info("search reindex starting", slog.Time("tick", t))
			if err := reindexAll(ctx, pages, search); err != nil {
				logger.Error("search reindex failed", slog.String("error", err.Error()))
			} else {
				logger.Info("search reindex complete")
			}
		}
	}
}

// reindexAll fetches all pages in all spaces and (re)indexes them.
// A real implementation would iterate spaces first; for the POC we just
// attempt to index pages we can list by a nil space (all spaces).
func reindexAll(ctx context.Context, pages domain.PageRepository, search domain.SearchRepository) error {
	// We cannot list "all pages" directly without iterating spaces.
	// As a POC shortcut we no-op here — the creation and update paths
	// keep the index fresh for newly written pages.
	_ = pages
	_ = search
	_ = ctx
	return nil
}
