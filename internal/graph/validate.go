package graph

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func (c *Client) ListDanglingPlaceholders(ctx context.Context) ([]EntitySummary, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n:_Placeholder) RETURN n", nil)
		if err != nil {
			return nil, err
		}
		var summaries []EntitySummary
		for res.Next(ctx) {
			value, _ := res.Record().Get("n")
			node, ok := value.(neo4j.Node)
			if !ok {
				continue
			}
			summaries = append(summaries, entitySummaryFromNode(node))
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return summaries, nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing dangling placeholders: %w", err)
	}

	return result.([]EntitySummary), nil
}

func (c *Client) ListOrphanedEntities(ctx context.Context) ([]EntitySummary, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n:Entity) WHERE NOT (n)--() RETURN n", nil)
		if err != nil {
			return nil, err
		}
		var summaries []EntitySummary
		for res.Next(ctx) {
			value, _ := res.Record().Get("n")
			node, ok := value.(neo4j.Node)
			if !ok {
				continue
			}
			summaries = append(summaries, entitySummaryFromNode(node))
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return summaries, nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing orphaned entities: %w", err)
	}

	return result.([]EntitySummary), nil
}

func (c *Client) ListDuplicateNames(ctx context.Context) ([]EntitySummary, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: c.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n:Entity) WITH n.layer AS layer, n.name_normalized AS name, count(*) AS c WHERE c > 1 RETURN layer, name", nil)
		if err != nil {
			return nil, err
		}
		var summaries []EntitySummary
		for res.Next(ctx) {
			layerValue, _ := res.Record().Get("layer")
			nameValue, _ := res.Record().Get("name")
			layer, _ := layerValue.(string)
			name, _ := nameValue.(string)
			summaries = append(summaries, EntitySummary{Name: name, Layer: layer})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return summaries, nil
	})
	if err != nil {
		return nil, fmt.Errorf("listing duplicate names: %w", err)
	}

	return result.([]EntitySummary), nil
}

func (c *Client) ListCrossLayerViolations(ctx context.Context) ([]EntitySummary, error) {
	// TODO: implement once event model and cross-layer rules are defined.
	return []EntitySummary{}, nil
}

func entitySummaryFromNode(node neo4j.Node) EntitySummary {
	props := node.Props
	return EntitySummary{
		Name:       toString(props["name"]),
		EntityType: toString(props["entity_type"]),
		Layer:      toString(props["layer"]),
		Tags:       toStringSlice(props["tags"]),
	}
}
