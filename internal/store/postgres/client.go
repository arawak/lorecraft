package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"lorecraft/internal/config"
	"lorecraft/internal/store"
)

var _ store.Store = (*Client)(nil)

type Client struct {
	pool *pgxpool.Pool
	cfg  *config.ProjectConfig
}

func New(ctx context.Context, dsn string, cfg *config.ProjectConfig) (*Client, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("creating postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging postgres: %w", err)
	}
	return &Client{pool: pool, cfg: cfg}, nil
}

func (c *Client) Close(ctx context.Context) error {
	c.pool.Close()
	return nil
}
