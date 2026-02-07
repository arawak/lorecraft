package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func (c *Client) FindEntityLayer(ctx context.Context, name string, layers []string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("graph client is nil")
	}
	if len(layers) == 0 {
		return "", nil
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	nameNormalized := strings.ToLower(name)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `UNWIND range(0, size($layers) - 1) AS idx
WITH idx, $layers[idx] AS layer
MATCH (n:Entity {name_normalized: $name_normalized, layer: layer})
RETURN layer
ORDER BY idx
LIMIT 1`, map[string]any{"layers": layers, "name_normalized": nameNormalized})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			value, _ := res.Record().Get("layer")
			layer, ok := value.(string)
			if ok {
				return layer, nil
			}
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return "", nil
	})
	if err != nil {
		return "", fmt.Errorf("find entity layer: %w", err)
	}

	return result.(string), nil
}
