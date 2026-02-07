package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	t.Run("valid config loads", func(t *testing.T) {
		cfg, err := LoadProjectConfig(filepath.Join("testdata", "valid_config.yaml"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cfg.Project != "test-project" {
			t.Fatalf("expected project name, got %q", cfg.Project)
		}
	})

	t.Run("missing project name", func(t *testing.T) {
		path := writeTempConfig(t, "version: 1\nneo4j:\n  uri: bolt://localhost:7687\nlayers:\n  - name: setting\n    paths: [./lore]\n")
		if _, err := LoadProjectConfig(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("missing neo4j uri", func(t *testing.T) {
		path := writeTempConfig(t, "project: test\nversion: 1\nneo4j:\n  uri: \nlayers:\n  - name: setting\n    paths: [./lore]\n")
		if _, err := LoadProjectConfig(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("no layers", func(t *testing.T) {
		path := writeTempConfig(t, "project: test\nversion: 1\nneo4j:\n  uri: bolt://localhost:7687\n")
		if _, err := LoadProjectConfig(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("layer missing name", func(t *testing.T) {
		path := writeTempConfig(t, "project: test\nversion: 1\nneo4j:\n  uri: bolt://localhost:7687\nlayers:\n  - paths: [./lore]\n")
		if _, err := LoadProjectConfig(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("layer missing paths", func(t *testing.T) {
		path := writeTempConfig(t, "project: test\nversion: 1\nneo4j:\n  uri: bolt://localhost:7687\nlayers:\n  - name: setting\n")
		if _, err := LoadProjectConfig(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("duplicate layer names", func(t *testing.T) {
		path := writeTempConfig(t, "project: test\nversion: 1\nneo4j:\n  uri: bolt://localhost:7687\nlayers:\n  - name: setting\n    paths: [./lore]\n  - name: Setting\n    paths: [./lore2]\n")
		if _, err := LoadProjectConfig(path); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("file not found", func(t *testing.T) {
		if _, err := LoadProjectConfig(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		path := writeTempConfig(t, "project: [\n")
		if _, err := LoadProjectConfig(path); err == nil {
			t.Fatalf("expected error")
		}
	})
}

func writeTempConfig(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing temp config: %v", err)
	}
	return path
}
