package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"lorecraft/internal/config"
)

type Client struct {
	driver   neo4j.DriverWithContext
	database string
}

func NewClient(ctx context.Context, uri, username, password, database string) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("creating neo4j driver: %w", err)
	}

	if err := driver.VerifyConnectivity(ctx); err != nil {
		_ = driver.Close(ctx)
		return nil, fmt.Errorf("verifying neo4j connectivity: %w", err)
	}

	return &Client{driver: driver, database: database}, nil
}

func (c *Client) Close(ctx context.Context) error {
	if c == nil || c.driver == nil {
		return nil
	}
	return c.driver.Close(ctx)
}

func (c *Client) EnsureIndexes(ctx context.Context, schema *config.Schema) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	statements := []string{
		`CREATE CONSTRAINT entity_unique_name_layer IF NOT EXISTS
FOR (e:Entity) REQUIRE (e.name_normalized, e.layer) IS UNIQUE`,
		`CREATE FULLTEXT INDEX entity_fulltext IF NOT EXISTS
FOR (e:Entity) ON EACH [e.name, e.tags_text]`,
		`CREATE INDEX entity_layer IF NOT EXISTS FOR (e:Entity) ON (e.layer)`,
		`CREATE INDEX entity_source_file IF NOT EXISTS FOR (e:Entity) ON (e.source_file)`,
	}

	for _, stmt := range statements {
		if _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			_, err := tx.Run(ctx, stmt, nil)
			return nil, err
		}); err != nil {
			return fmt.Errorf("ensuring indexes: %w", err)
		}
	}

	return nil
}
