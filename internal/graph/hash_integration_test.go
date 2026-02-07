//go:build integration

package graph

import (
	"context"
	"testing"
)

func TestGetLayerHashes(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	inputs := []EntityInput{
		{
			Name:       "Alpha",
			EntityType: "npc",
			Label:      "NPC",
			Layer:      "setting",
			SourceFile: "lore/alpha.md",
			SourceHash: "hash-alpha",
		},
		{
			Name:       "Beta",
			EntityType: "npc",
			Label:      "NPC",
			Layer:      "setting",
			SourceFile: "lore/beta.md",
			SourceHash: "hash-beta",
		},
	}

	for _, input := range inputs {
		if err := client.UpsertEntity(ctx, input); err != nil {
			t.Fatalf("upsert entity: %v", err)
		}
	}

	hashes, err := client.GetLayerHashes(ctx, "setting")
	if err != nil {
		t.Fatalf("get layer hashes: %v", err)
	}
	if len(hashes) != 2 {
		t.Fatalf("expected 2 hashes, got %d", len(hashes))
	}
	if hashes["lore/alpha.md"] != "hash-alpha" {
		t.Fatalf("expected hash-alpha, got %q", hashes["lore/alpha.md"])
	}
	if hashes["lore/beta.md"] != "hash-beta" {
		t.Fatalf("expected hash-beta, got %q", hashes["lore/beta.md"])
	}
}
