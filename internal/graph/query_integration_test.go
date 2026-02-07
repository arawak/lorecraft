//go:build integration

package graph

import (
	"context"
	"testing"

	"lorecraft/internal/config"
)

func TestGetEntity_Found(t *testing.T) {
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

	entity, err := client.GetEntity(ctx, "Test NPC", "")
	if err != nil {
		t.Fatalf("get entity: %v", err)
	}
	if entity == nil {
		t.Fatalf("expected entity")
	}

	if entity.Name != "Test NPC" {
		t.Fatalf("expected name, got %q", entity.Name)
	}
	if entity.EntityType != "npc" {
		t.Fatalf("expected type, got %q", entity.EntityType)
	}
	if entity.Layer != "setting" {
		t.Fatalf("expected layer, got %q", entity.Layer)
	}
	if entity.SourceFile != "lore/test.md" {
		t.Fatalf("expected source file, got %q", entity.SourceFile)
	}
	if entity.SourceHash != "hash" {
		t.Fatalf("expected source hash, got %q", entity.SourceHash)
	}
	if len(entity.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(entity.Tags))
	}
	if entity.Properties["role"] != "Guard" {
		t.Fatalf("expected role property")
	}
	if _, ok := entity.Properties["name"]; ok {
		t.Fatalf("did not expect standard properties in Properties map")
	}
}

func TestGetEntity_NotFound(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	entity, err := client.GetEntity(ctx, "Missing", "")
	if err != nil {
		t.Fatalf("get entity: %v", err)
	}
	if entity != nil {
		t.Fatalf("expected nil entity")
	}
}

func TestGetEntity_Ambiguous(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	if err := client.UpsertEntity(ctx, EntityInput{Name: "Same", EntityType: "npc", Label: "NPC", Layer: "setting"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}
	if err := client.UpsertEntity(ctx, EntityInput{Name: "Same", EntityType: "npc", Label: "NPC", Layer: "campaign"}); err != nil {
		t.Fatalf("upsert entity: %v", err)
	}

	_, err := client.GetEntity(ctx, "Same", "")
	if err == nil {
		t.Fatalf("expected ambiguity error")
	}
}

func TestListEntities_Filters(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	inputs := []EntityInput{
		{Name: "A", EntityType: "npc", Label: "NPC", Layer: "setting", Tags: []string{"alpha"}},
		{Name: "B", EntityType: "faction", Label: "FACTION", Layer: "setting", Tags: []string{"beta"}},
		{Name: "C", EntityType: "npc", Label: "NPC", Layer: "campaign", Tags: []string{"alpha", "beta"}},
	}
	for _, input := range inputs {
		if err := client.UpsertEntity(ctx, input); err != nil {
			t.Fatalf("upsert entity: %v", err)
		}
	}

	byType, err := client.ListEntities(ctx, "npc", "", "")
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}
	if len(byType) != 2 {
		t.Fatalf("expected 2 npc entities, got %d", len(byType))
	}

	byLayer, err := client.ListEntities(ctx, "", "setting", "")
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}
	if len(byLayer) != 2 {
		t.Fatalf("expected 2 setting entities, got %d", len(byLayer))
	}

	byTag, err := client.ListEntities(ctx, "", "", "alpha")
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}
	if len(byTag) != 2 {
		t.Fatalf("expected 2 alpha entities, got %d", len(byTag))
	}

	byTypeLayer, err := client.ListEntities(ctx, "npc", "setting", "")
	if err != nil {
		t.Fatalf("list entities: %v", err)
	}
	if len(byTypeLayer) != 1 || byTypeLayer[0].Name != "A" {
		t.Fatalf("expected only A, got %+v", byTypeLayer)
	}
}

func TestGetRelationships_Directions(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	inputs := []EntityInput{
		{Name: "A", EntityType: "npc", Label: "NPC", Layer: "setting"},
		{Name: "B", EntityType: "npc", Label: "NPC", Layer: "setting"},
		{Name: "C", EntityType: "npc", Label: "NPC", Layer: "setting"},
	}
	for _, input := range inputs {
		if err := client.UpsertEntity(ctx, input); err != nil {
			t.Fatalf("upsert entity: %v", err)
		}
	}
	if err := client.UpsertRelationship(ctx, "A", "setting", "B", "setting", "RELATED_TO"); err != nil {
		t.Fatalf("upsert relationship: %v", err)
	}
	if err := client.UpsertRelationship(ctx, "B", "setting", "C", "setting", "RELATED_TO"); err != nil {
		t.Fatalf("upsert relationship: %v", err)
	}

	outgoing, err := client.GetRelationships(ctx, "A", "RELATED_TO", "outgoing", 1)
	if err != nil {
		t.Fatalf("get relationships: %v", err)
	}
	if len(outgoing) != 1 {
		t.Fatalf("expected 1 outgoing relationship, got %d", len(outgoing))
	}
	if outgoing[0].Direction != "outgoing" {
		t.Fatalf("expected outgoing direction, got %q", outgoing[0].Direction)
	}
	if outgoing[0].From.Name != "A" || outgoing[0].To.Name != "B" {
		t.Fatalf("unexpected relationship endpoints: %+v", outgoing[0])
	}

	incoming, err := client.GetRelationships(ctx, "A", "RELATED_TO", "incoming", 1)
	if err != nil {
		t.Fatalf("get relationships: %v", err)
	}
	if len(incoming) != 0 {
		t.Fatalf("expected 0 incoming relationships, got %d", len(incoming))
	}

	both, err := client.GetRelationships(ctx, "A", "RELATED_TO", "both", 2)
	if err != nil {
		t.Fatalf("get relationships: %v", err)
	}
	if len(both) < 2 {
		t.Fatalf("expected at least 2 relationships, got %d", len(both))
	}
	var sawC bool
	for _, rel := range both {
		if rel.To.Name == "C" {
			sawC = true
		}
	}
	if !sawC {
		t.Fatalf("expected to see relationship to C")
	}
}

func TestGetRelationships_InvalidInputs(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	if _, err := client.GetRelationships(ctx, "A", "RELATED_TO", "sideways", 1); err == nil {
		t.Fatalf("expected direction error")
	}
	if _, err := client.GetRelationships(ctx, "A", "bad-type", "both", 1); err == nil {
		t.Fatalf("expected relType error")
	}
	if _, err := client.GetRelationships(ctx, "A", "RELATED_TO", "both", 0); err == nil {
		t.Fatalf("expected depth error")
	}
	if _, err := client.GetRelationships(ctx, "A", "RELATED_TO", "both", 6); err == nil {
		t.Fatalf("expected depth error")
	}
}

func TestSearch(t *testing.T) {
	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	if err := client.EnsureIndexes(ctx, &config.Schema{Version: 1}); err != nil {
		t.Fatalf("ensure indexes: %v", err)
	}

	inputs := []EntityInput{
		{Name: "Bureau of Civic Affairs", EntityType: "faction", Label: "FACTION", Layer: "setting", Tags: []string{"bureaucracy"}},
		{Name: "Westport", EntityType: "settlement", Label: "SETTLEMENT", Layer: "setting", Tags: []string{"coastal"}},
	}
	for _, input := range inputs {
		if err := client.UpsertEntity(ctx, input); err != nil {
			t.Fatalf("upsert entity: %v", err)
		}
	}

	results, err := client.Search(ctx, "bureau", "", "")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected search results")
	}
	if results[0].Name == "" {
		t.Fatalf("expected result names")
	}
	var found bool
	for _, result := range results {
		if result.Name == "Bureau of Civic Affairs" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Bureau of Civic Affairs in results")
	}
}
