package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func (c *Client) GetLayerHashes(ctx context.Context, layer string) (map[string]string, error) {
	if c == nil {
		return nil, fmt.Errorf("graph client is nil")
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `MATCH (n:Entity {layer: $layer})
WHERE n.source_file IS NOT NULL AND n._placeholder IS NULL
RETURN n.source_file AS source_file, n.source_hash AS source_hash`, map[string]any{"layer": layer})
		if err != nil {
			return nil, err
		}
		values := make(map[string]string)
		for res.Next(ctx) {
			record := res.Record()
			sourceFileValue, _ := record.Get("source_file")
			sourceFile, ok := sourceFileValue.(string)
			if !ok || sourceFile == "" {
				continue
			}
			sourceHashValue, _ := record.Get("source_hash")
			sourceHash, ok := sourceHashValue.(string)
			if !ok {
				sourceHash = ""
			}
			values[sourceFile] = sourceHash
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return values, nil
	})
	if err != nil {
		return nil, fmt.Errorf("query layer hashes: %w", err)
	}

	return result.(map[string]string), nil
}
