package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func (c *Client) RunCypher(ctx context.Context, query string, params map[string]any) ([]map[string]any, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0)
		for res.Next(ctx) {
			record := res.Record()
			row := make(map[string]any, len(record.Keys))
			for _, key := range record.Keys {
				value, _ := record.Get(key)
				row[key] = value
			}
			rows = append(rows, row)
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return rows, nil
	})
	if err != nil {
		return nil, fmt.Errorf("run cypher: %w", err)
	}

	return result.([]map[string]any), nil
}
