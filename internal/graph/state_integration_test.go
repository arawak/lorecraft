//go:build integration

package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGetCurrentState(t *testing.T) {
	restore := writeTempConfig(t)
	defer restore()

	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	base := EntityInput{
		Name:       "Westport",
		EntityType: "settlement",
		Label:      "SETTLEMENT",
		Layer:      "setting",
		SourceFile: "lore/westport.md",
		SourceHash: "hash",
		Properties: map[string]any{"status": "intact", "features": []string{"coastal"}},
	}
	if err := client.UpsertEntity(ctx, base); err != nil {
		t.Fatalf("upsert base entity: %v", err)
	}

	npc := EntityInput{
		Name:       "Test NPC",
		EntityType: "npc",
		Label:      "NPC",
		Layer:      "setting",
		SourceFile: "lore/npc.md",
		SourceHash: "hash",
	}
	if err := client.UpsertEntity(ctx, npc); err != nil {
		t.Fatalf("upsert npc: %v", err)
	}

	consequence1 := []Consequence{{Entity: "Westport", Property: "status", Value: "damaged"}}
	payload1, err := json.Marshal(consequence1)
	if err != nil {
		t.Fatalf("marshal consequence: %v", err)
	}

	event1 := EntityInput{
		Name:       "Storm Surge",
		EntityType: "event",
		Label:      "EVENT",
		Layer:      "campaign",
		SourceFile: "campaign/event1.md",
		SourceHash: "hash",
		Properties: map[string]any{"session": 1, "consequences_json": string(payload1)},
	}
	if err := client.UpsertEntity(ctx, event1); err != nil {
		t.Fatalf("upsert event1: %v", err)
	}
	if err := client.UpsertRelationship(ctx, event1.Name, event1.Layer, base.Name, base.Layer, "AFFECTS"); err != nil {
		t.Fatalf("upsert affects: %v", err)
	}
	if err := client.UpsertRelationship(ctx, event1.Name, event1.Layer, npc.Name, npc.Layer, "INVOLVES"); err != nil {
		t.Fatalf("upsert involves: %v", err)
	}
	if err := client.UpsertRelationship(ctx, event1.Name, event1.Layer, base.Name, base.Layer, "OCCURS_IN"); err != nil {
		t.Fatalf("upsert occurs in: %v", err)
	}

	consequence2 := []Consequence{{Entity: "Westport", Property: "features", Add: "rebuilt"}}
	payload2, err := json.Marshal(consequence2)
	if err != nil {
		t.Fatalf("marshal consequence: %v", err)
	}

	event2 := EntityInput{
		Name:       "Reconstruction",
		EntityType: "event",
		Label:      "EVENT",
		Layer:      "campaign",
		SourceFile: "campaign/event2.md",
		SourceHash: "hash",
		Properties: map[string]any{"session": 2, "consequences_json": string(payload2)},
	}
	if err := client.UpsertEntity(ctx, event2); err != nil {
		t.Fatalf("upsert event2: %v", err)
	}
	if err := client.UpsertRelationship(ctx, event2.Name, event2.Layer, base.Name, base.Layer, "AFFECTS"); err != nil {
		t.Fatalf("upsert affects: %v", err)
	}

	state, err := client.GetCurrentState(ctx, "Westport", "campaign")
	if err != nil {
		t.Fatalf("get current state: %v", err)
	}
	if state == nil {
		t.Fatalf("expected current state")
	}

	if state.BaseProperties["status"] != "intact" {
		t.Fatalf("expected base status intact")
	}
	if state.CurrentProperties["status"] != "damaged" {
		t.Fatalf("expected current status damaged, got %v", state.CurrentProperties["status"])
	}

	tags := toStringList(state.CurrentProperties["features"])
	if len(tags) != 2 {
		t.Fatalf("expected features list, got %#v", state.CurrentProperties["features"])
	}
	if tags[1] != "rebuilt" {
		t.Fatalf("expected rebuilt tag, got %v", tags)
	}

	if len(state.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(state.Events))
	}
	if state.Events[0].Name != "Storm Surge" || state.Events[0].Session != 1 {
		t.Fatalf("unexpected first event: %#v", state.Events[0])
	}
	if len(state.Events[0].Participants) != 1 || state.Events[0].Participants[0] != "Test NPC" {
		t.Fatalf("expected participants for event1")
	}
	if len(state.Events[0].Location) != 1 || state.Events[0].Location[0] != "Westport" {
		t.Fatalf("expected location for event1")
	}
}

func TestGetTimeline(t *testing.T) {
	restore := writeTempConfig(t)
	defer restore()

	ctx := context.Background()
	client := testClient(t)
	clearDatabase(t, client)

	base := EntityInput{
		Name:       "Westport",
		EntityType: "settlement",
		Label:      "SETTLEMENT",
		Layer:      "setting",
		SourceFile: "lore/westport.md",
		SourceHash: "hash",
	}
	if err := client.UpsertEntity(ctx, base); err != nil {
		t.Fatalf("upsert base entity: %v", err)
	}

	other := EntityInput{
		Name:       "Iron Tide",
		EntityType: "faction",
		Label:      "FACTION",
		Layer:      "setting",
		SourceFile: "lore/faction.md",
		SourceHash: "hash",
	}
	if err := client.UpsertEntity(ctx, other); err != nil {
		t.Fatalf("upsert other entity: %v", err)
	}

	event1 := EntityInput{
		Name:       "Storm Surge",
		EntityType: "event",
		Label:      "EVENT",
		Layer:      "campaign",
		SourceFile: "campaign/event1.md",
		SourceHash: "hash",
		Properties: map[string]any{"session": 1},
	}
	if err := client.UpsertEntity(ctx, event1); err != nil {
		t.Fatalf("upsert event1: %v", err)
	}
	if err := client.UpsertRelationship(ctx, event1.Name, event1.Layer, base.Name, base.Layer, "AFFECTS"); err != nil {
		t.Fatalf("upsert affects: %v", err)
	}

	event2 := EntityInput{
		Name:       "Raid",
		EntityType: "event",
		Label:      "EVENT",
		Layer:      "campaign",
		SourceFile: "campaign/event2.md",
		SourceHash: "hash",
		Properties: map[string]any{"session": 2},
	}
	if err := client.UpsertEntity(ctx, event2); err != nil {
		t.Fatalf("upsert event2: %v", err)
	}
	if err := client.UpsertRelationship(ctx, event2.Name, event2.Layer, other.Name, other.Layer, "AFFECTS"); err != nil {
		t.Fatalf("upsert affects: %v", err)
	}

	events, err := client.GetTimeline(ctx, "campaign", "Westport", 0, 0)
	if err != nil {
		t.Fatalf("get timeline: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Name != "Storm Surge" {
		t.Fatalf("unexpected event: %#v", events[0])
	}

	events, err = client.GetTimeline(ctx, "campaign", "", 2, 2)
	if err != nil {
		t.Fatalf("get timeline range: %v", err)
	}
	if len(events) != 1 || events[0].Name != "Raid" {
		t.Fatalf("expected Raid event, got %#v", events)
	}
}

func writeTempConfig(t *testing.T) func() {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "lorecraft.yaml")
	configContents := []byte("project: test\nversion: 1\nneo4j:\n  uri: bolt://localhost:7687\n  username: neo4j\n  password: changeme\n  database: neo4j\nlayers:\n  - name: setting\n    paths: [./lore]\n    canonical: true\n  - name: campaign\n    paths: [./campaign]\n    canonical: false\n    depends_on: [setting]\n")
	if err := os.WriteFile(configPath, configContents, 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp config dir: %v", err)
	}
	return func() {
		_ = os.Chdir(cwd)
	}
}

func toStringList(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
