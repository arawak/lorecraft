package parser

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("valid npc with full frontmatter", func(t *testing.T) {
		content := []byte("---\ntitle: Test NPC\ntype: npc\nrole: Guard Captain\nstatus: alive\nlocation: Testville\nfaction: The Watch\ntags: [military, law-enforcement]\nrelated: [Mayor Teston]\n---\n\nThis is the body describing the NPC.\n")
		doc, err := Parse(content)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if doc.Title != "Test NPC" {
			t.Fatalf("expected title, got %q", doc.Title)
		}
		if doc.EntityType != "npc" {
			t.Fatalf("expected type npc, got %q", doc.EntityType)
		}
		if doc.Body == "" {
			t.Fatalf("expected body")
		}
		if !reflect.DeepEqual(doc.Tags, []string{"military", "law-enforcement"}) {
			t.Fatalf("unexpected tags: %#v", doc.Tags)
		}
		if _, ok := doc.Frontmatter["role"]; !ok {
			t.Fatalf("expected role in frontmatter")
		}
		if _, ok := doc.Frontmatter["related"]; !ok {
			t.Fatalf("expected related in frontmatter")
		}
	})

	t.Run("minimal frontmatter", func(t *testing.T) {
		content := []byte("---\ntitle: Minimal\ntype: lore\n---\n")
		doc, err := Parse(content)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if doc.Tags != nil {
			t.Fatalf("expected nil tags, got %#v", doc.Tags)
		}
		if doc.Body != "" {
			t.Fatalf("expected empty body, got %q", doc.Body)
		}
	})

	t.Run("no frontmatter", func(t *testing.T) {
		_, err := Parse([]byte("Just text"))
		if !errors.Is(err, ErrNoFrontmatter) {
			t.Fatalf("expected ErrNoFrontmatter, got %v", err)
		}
	})

	t.Run("missing closing marker", func(t *testing.T) {
		_, err := Parse([]byte("---\ntitle: Missing\n"))
		if !errors.Is(err, ErrNoFrontmatter) {
			t.Fatalf("expected ErrNoFrontmatter, got %v", err)
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		_, err := Parse([]byte("---\ntitle: [\n---\n"))
		if !errors.Is(err, ErrInvalidYAML) {
			t.Fatalf("expected ErrInvalidYAML, got %v", err)
		}
	})

	t.Run("missing title", func(t *testing.T) {
		_, err := Parse([]byte("---\ntype: npc\n---\n"))
		if !errors.Is(err, ErrMissingTitle) {
			t.Fatalf("expected ErrMissingTitle, got %v", err)
		}
	})

	t.Run("missing type", func(t *testing.T) {
		_, err := Parse([]byte("---\ntitle: Something\n---\n"))
		if !errors.Is(err, ErrMissingType) {
			t.Fatalf("expected ErrMissingType, got %v", err)
		}
	})

	t.Run("tags list", func(t *testing.T) {
		doc, err := Parse([]byte("---\ntitle: Tags\ntype: npc\ntags: [a, b]\n---\n"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !reflect.DeepEqual(doc.Tags, []string{"a", "b"}) {
			t.Fatalf("unexpected tags: %#v", doc.Tags)
		}
	})

	t.Run("tags single string", func(t *testing.T) {
		doc, err := Parse([]byte("---\ntitle: Tags\ntype: npc\ntags: lone\n---\n"))
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !reflect.DeepEqual(doc.Tags, []string{"lone"}) {
			t.Fatalf("unexpected tags: %#v", doc.Tags)
		}
	})
}

func TestParseFile(t *testing.T) {
	doc, err := ParseFile(filepath.Join("testdata", "valid_npc.md"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if doc.Title != "Test NPC" {
		t.Fatalf("expected title, got %q", doc.Title)
	}
	if doc.SourceFile == "" {
		t.Fatalf("expected source file set")
	}
}

func TestParseFile_NoFrontmatter(t *testing.T) {
	_, err := ParseFile(filepath.Join("testdata", "no_frontmatter.md"))
	if !errors.Is(err, ErrNoFrontmatter) {
		t.Fatalf("expected ErrNoFrontmatter, got %v", err)
	}
}

func TestParseFile_MissingType(t *testing.T) {
	_, err := ParseFile(filepath.Join("testdata", "missing_type.md"))
	if !errors.Is(err, ErrMissingType) {
		t.Fatalf("expected ErrMissingType, got %v", err)
	}
}

func TestParse_BOMTrim(t *testing.T) {
	content := []byte("\ufeff---\ntitle: BOM\ntype: npc\n---\n")
	doc, err := Parse(content)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if doc.Title != "BOM" {
		t.Fatalf("expected title, got %q", doc.Title)
	}
}

func TestParseFile_ReadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.md")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected missing file")
	}
	if _, err := ParseFile(path); err == nil {
		t.Fatalf("expected error")
	}
}
