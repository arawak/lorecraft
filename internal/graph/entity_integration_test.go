//go:build integration

package graph

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestUpsertEntity(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	input := EntityInput{
		Name:       "Test NPC",
		EntityType: "npc",
		Label:      "NPC",
		Layer:      "setting",
		SourceFile: "lore/test.md",
		SourceHash: "hash",
		Properties: map[string]any{"role": "Guard"},
		Tags:       []string{"tag1", "tag2"},
	}

	if err := client.UpsertEntity(ctx, input); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}

	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n:Entity {name_normalized: $nn, layer: $layer}) RETURN n", map[string]any{"nn": "test npc", "layer": "setting"})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			value, _ := res.Record().Get("n")
			return value.(neo4j.Node), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("query node: %v", err)
	}

	node := result.(neo4j.Node)
	if node.Props["name"] != "Test NPC" {
		t.Fatalf("expected name, got %v", node.Props["name"])
	}
	if node.Props["entity_type"] != "npc" {
		t.Fatalf("expected entity_type, got %v", node.Props["entity_type"])
	}
	if node.Props["role"] != "Guard" {
		t.Fatalf("expected role property, got %v", node.Props["role"])
	}
	if !hasLabel(node.Labels, "Entity") || !hasLabel(node.Labels, "NPC") {
		t.Fatalf("expected Entity and NPC labels")
	}
}

func TestUpsertEntity_Idempotent(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	input := EntityInput{
		Name:       "Test NPC",
		EntityType: "npc",
		Label:      "NPC",
		Layer:      "setting",
		SourceFile: "lore/test.md",
		SourceHash: "hash",
		Properties: map[string]any{"role": "Guard"},
	}

	if err := client.UpsertEntity(ctx, input); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}
	if err := client.UpsertEntity(ctx, input); err != nil {
		t.Fatalf("upsert entity again: %v", err)
	}

	count := countNodes(t, client)
	if count != 1 {
		t.Fatalf("expected 1 node, got %d", count)
	}
}

func TestUpsertEntity_UpdateProperties(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	input := EntityInput{
		Name:       "Test NPC",
		EntityType: "npc",
		Label:      "NPC",
		Layer:      "setting",
		SourceFile: "lore/test.md",
		SourceHash: "hash",
		Properties: map[string]any{"role": "Guard"},
	}

	if err := client.UpsertEntity(ctx, input); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}
	input.Properties["role"] = "Captain"
	if err := client.UpsertEntity(ctx, input); err != nil {
		t.Fatalf("upsert entity update: %v", err)
	}

	role := getNodeProperty(t, client, "test npc", "setting", "role")
	if role != "Captain" {
		t.Fatalf("expected updated role, got %v", role)
	}
}

func TestUpsertEntity_InvalidLabel(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)

	input := EntityInput{Label: "bad-label"}
	if err := client.UpsertEntity(ctx, input); err == nil {
		t.Fatalf("expected error")
	}
}

func clearDatabase(t *testing.T, client *Client) {
	t.Helper()
	ctx := context.Background()
	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)
	if _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
		return nil, err
	}); err != nil {
		t.Fatalf("clearing database: %v", err)
	}
}

func countNodes(t *testing.T, client *Client) int64 {
	t.Helper()
	ctx := context.Background()
	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n) RETURN count(n) AS count", nil)
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			value, _ := res.Record().Get("count")
			return value.(int64), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return int64(0), nil
	})
	if err != nil {
		t.Fatalf("counting nodes: %v", err)
	}
	return result.(int64)
}

func getNodeProperty(t *testing.T, client *Client, nameNormalized, layer, prop string) any {
	t.Helper()
	ctx := context.Background()
	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n:Entity {name_normalized: $nn, layer: $layer}) RETURN n", map[string]any{"nn": nameNormalized, "layer": layer})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			value, _ := res.Record().Get("n")
			return value.(neo4j.Node), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("query node: %v", err)
	}
	node := result.(neo4j.Node)
	return node.Props[prop]
}

func hasLabel(labels []string, target string) bool {
	for _, label := range labels {
		if label == target {
			return true
		}
	}
	return false
}
