//go:build integration

package graph

import (
	"context"
	"testing"
)

func TestRemoveStaleNodes(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	inputs := []EntityInput{
		{Name: "A", EntityType: "npc", Label: "NPC", Layer: "setting", SourceFile: "lore/a.md"},
		{Name: "B", EntityType: "npc", Label: "NPC", Layer: "setting", SourceFile: "lore/b.md"},
		{Name: "C", EntityType: "npc", Label: "NPC", Layer: "setting", SourceFile: "lore/c.md"},
	}
	for _, input := range inputs {
		if err := client.UpsertEntity(ctx, input); err != nil {
			t.Fatalf("upsert entity: %v", err)
		}
	}

	deleted, err := client.RemoveStaleNodes(ctx, "setting", []string{"lore/a.md", "lore/c.md"})
	if err != nil {
		t.Fatalf("remove stale nodes: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted, got %d", deleted)
	}
}

func TestRemoveStaleNodes_PreservesPlaceholders(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	if err := client.UpsertEntity(ctx, EntityInput{Name: "A", EntityType: "npc", Label: "NPC", Layer: "setting", SourceFile: "lore/a.md"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}
	if err := client.UpsertRelationship(ctx, "A", "setting", "Missing", "setting", "RELATED_TO"); err != nil {
		t.Fatalf("upsert relationship: %v", err)
	}

	deleted, err := client.RemoveStaleNodes(ctx, "setting", []string{"lore/a.md"})
	if err != nil {
		t.Fatalf("remove stale nodes: %v", err)
	}
	if deleted != 0 {
		t.Fatalf("expected 0 deleted, got %d", deleted)
	}
}
