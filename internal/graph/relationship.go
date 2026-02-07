package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func (c *Client) UpsertRelationship(ctx context.Context, fromName, fromLayer, toName, toLayer, relType string) error {
	if strings.TrimSpace(relType) == "" || !labelPattern.MatchString(relType) {
		return fmt.Errorf("invalid relationship type: %s", relType)
	}

	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	query := fmt.Sprintf(`
MATCH (a:Entity {name_normalized: $from_nn, layer: $from_layer})
MERGE (b:Entity {name_normalized: $to_nn, layer: $to_layer})
ON CREATE SET b.name = $to_name, b._placeholder = true, b:_Placeholder
MERGE (a)-[r:%s]->(b)
`, relType)

	params := map[string]any{
		"from_nn":    strings.ToLower(fromName),
		"from_layer": fromLayer,
		"to_nn":      strings.ToLower(toName),
		"to_layer":   toLayer,
		"to_name":    toName,
	}

	if _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	}); err != nil {
		return fmt.Errorf("upserting relationship: %w", err)
	}

	return nil
}
