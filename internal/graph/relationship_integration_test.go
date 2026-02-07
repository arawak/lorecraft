//go:build integration

package graph

import (
	"context"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestUpsertRelationship(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	if err := client.UpsertEntity(ctx, EntityInput{Name: "A", EntityType: "npc", Label: "NPC", Layer: "setting"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}
	if err := client.UpsertEntity(ctx, EntityInput{Name: "B", EntityType: "npc", Label: "NPC", Layer: "setting"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}

	if err := client.UpsertRelationship(ctx, "A", "setting", "B", "setting", "RELATED_TO"); err != nil {
		t.Fatalf("upsert relationship: %v", err)
	}

	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)

	count := countRelationships(t, session)
	if count != 1 {
		t.Fatalf("expected 1 relationship, got %d", count)
	}
}

func TestUpsertRelationship_Placeholder(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	if err := client.UpsertEntity(ctx, EntityInput{Name: "A", EntityType: "npc", Label: "NPC", Layer: "setting"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}

	if err := client.UpsertRelationship(ctx, "A", "setting", "Missing", "setting", "RELATED_TO"); err != nil {
		t.Fatalf("upsert relationship: %v", err)
	}

	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)

	placeholderCount, err := countPlaceholders(ctx, session)
	if err != nil {
		t.Fatalf("count placeholders: %v", err)
	}
	if placeholderCount != 1 {
		t.Fatalf("expected 1 placeholder, got %d", placeholderCount)
	}
}

func TestUpsertRelationship_Idempotent(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	if err := client.UpsertEntity(ctx, EntityInput{Name: "A", EntityType: "npc", Label: "NPC", Layer: "setting"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}
	if err := client.UpsertEntity(ctx, EntityInput{Name: "B", EntityType: "npc", Label: "NPC", Layer: "setting"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}

	if err := client.UpsertRelationship(ctx, "A", "setting", "B", "setting", "RELATED_TO"); err != nil {
		t.Fatalf("upsert relationship: %v", err)
	}
	if err := client.UpsertRelationship(ctx, "A", "setting", "B", "setting", "RELATED_TO"); err != nil {
		t.Fatalf("upsert relationship: %v", err)
	}

	session := client.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: client.database})
	defer session.Close(ctx)

	count := countRelationships(t, session)
	if count != 1 {
		t.Fatalf("expected 1 relationship, got %d", count)
	}
}

func TestUpsertRelationship_InvalidRelType(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)

	if err := client.UpsertRelationship(ctx, "A", "setting", "B", "setting", "bad-type"); err == nil {
		t.Fatalf("expected error")
	}
}

func countRelationships(t *testing.T, session neo4j.SessionWithContext) int64 {
	t.Helper()
	ctx := context.Background()
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH ()-[r]->() RETURN count(r) AS count", nil)
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
		t.Fatalf("count relationships: %v", err)
	}
	return result.(int64)
}

func countPlaceholders(ctx context.Context, session neo4j.SessionWithContext) (int64, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n:_Placeholder) RETURN count(n) AS count", nil)
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
		return 0, err
	}
	return result.(int64), nil
}
