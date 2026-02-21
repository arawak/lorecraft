package main

import (
	"context"

	"lorecraft/internal/config"
	"lorecraft/internal/store/postgres"
)

func openDB(ctx context.Context, cfg *config.ProjectConfig) (*postgres.Client, error) {
	return postgres.New(ctx, cfg.Database.DSN, cfg)
}
