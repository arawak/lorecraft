package main

import (
	"context"
	"strings"

	"lorecraft/internal/config"
	"lorecraft/internal/store"
	"lorecraft/internal/store/postgres"
	"lorecraft/internal/store/sqlite"
)

func openDB(ctx context.Context, cfg *config.ProjectConfig) (store.Store, error) {
	dsn := cfg.Database.DSN
	switch {
	case strings.HasPrefix(dsn, "postgres://"):
		return postgres.New(ctx, dsn, cfg)
	case strings.HasPrefix(dsn, "sqlite://"):
		return sqlite.New(ctx, dsn, cfg)
	default:
		return nil, nil
	}
}
