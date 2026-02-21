package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"lorecraft/internal/config"
	"lorecraft/internal/store"

	_ "modernc.org/sqlite"
)

var _ store.Store = (*Client)(nil)

type Client struct {
	db  *sql.DB
	cfg *config.ProjectConfig
}

func New(ctx context.Context, dsn string, cfg *config.ProjectConfig) (*Client, error) {
	driverDSN, err := parseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing sqlite DSN: %w", err)
	}

	db, err := sql.Open("sqlite", driverDSN)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite database: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("pinging sqlite: %w", err)
	}

	pragmas := []string{
		"PRAGMA busy_timeout = 30000;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", pragma, err)
		}
	}

	return &Client{db: db, cfg: cfg}, nil
}

func (c *Client) Close(ctx context.Context) error {
	return c.db.Close()
}
