package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func (c *Client) RemoveStaleNodes(ctx context.Context, layer string, currentSourceFiles []string) (int64, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := `
MATCH (n:Entity {layer: $layer})
WHERE n.source_file IS NOT NULL
  AND NOT n.source_file IN $current_files
  AND n._placeholder IS NULL
DETACH DELETE n
RETURN count(n) AS deleted
`

	params := map[string]any{
		"layer":         layer,
		"current_files": currentSourceFiles,
	}

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			value, _ := res.Record().Get("deleted")
			if count, ok := value.(int64); ok {
				return count, nil
			}
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return int64(0), nil
	})
	if err != nil {
		return 0, fmt.Errorf("removing stale nodes: %w", err)
	}

	return result.(int64), nil
}
