package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSchema(t *testing.T) {
	t.Run("valid schema loads", func(t *testing.T) {
		schema, err := LoadSchema(filepath.Join("testdata", "valid_schema.yaml"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !schema.IsValidEntityType("npc") {
			t.Fatalf("expected npc entity type to be valid")
		}
	})

	t.Run("missing entity types", func(t *testing.T) {
		path := writeTempSchema(t, "version: 1\nentity_types: []\nrelationship_types: []\n")
		if _, err := LoadSchema(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("duplicate entity type names", func(t *testing.T) {
		path := writeTempSchema(t, "version: 1\nentity_types:\n  - name: npc\n  - name: NPC\nrelationship_types:\n  - name: RELATED_TO\n")
		if _, err := LoadSchema(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("enum property without values", func(t *testing.T) {
		path := writeTempSchema(t, "version: 1\nentity_types:\n  - name: npc\n    properties:\n      - { name: status, type: enum }\nrelationship_types:\n  - name: RELATED_TO\n")
		if _, err := LoadSchema(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("field mapping references unknown relationship", func(t *testing.T) {
		path := writeTempSchema(t, "version: 1\nentity_types:\n  - name: npc\n    field_mappings:\n      - { field: faction, relationship: MEMBER_OF }\nrelationship_types:\n  - name: RELATED_TO\n")
		if _, err := LoadSchema(path); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func TestSchemaHelpers(t *testing.T) {
	schema, err := LoadSchema(filepath.Join("testdata", "valid_schema.yaml"))
	if err != nil {
		t.Fatalf("loading schema: %v", err)
	}

	t.Run("EntityTypeByName case-insensitive", func(t *testing.T) {
		if _, ok := schema.EntityTypeByName("NPC"); !ok {
			t.Fatalf("expected to find NPC entity type")
		}
	})

	t.Run("NodeLabel uppercase", func(t *testing.T) {
		if label := schema.NodeLabel("npc"); label != "NPC" {
			t.Fatalf("expected NPC, got %q", label)
		}
	})

	t.Run("IsValidEntityType", func(t *testing.T) {
		if !schema.IsValidEntityType("npc") {
			t.Fatalf("expected npc to be valid")
		}
		if schema.IsValidEntityType("dragon") {
			t.Fatalf("expected dragon to be invalid")
		}
	})
}

func writeTempSchema(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing temp schema: %v", err)
	}
	return path
}
