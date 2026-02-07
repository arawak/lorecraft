package parser

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Document struct {
	Frontmatter map[string]any
	Title       string
	EntityType  string
	Tags        []string
	Body        string
	SourceFile  string
}

var (
	ErrNoFrontmatter = errors.New("no frontmatter found")
	ErrInvalidYAML   = errors.New("invalid YAML in frontmatter")
	ErrMissingTitle  = errors.New("frontmatter missing required 'title' field")
	ErrMissingType   = errors.New("frontmatter missing required 'type' field")
)

func ParseFile(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	doc, err := Parse(data)
	if err != nil {
		return nil, err
	}
	doc.SourceFile = path
	return doc, nil
}

func Parse(content []byte) (*Document, error) {
	trimmed := bytes.TrimLeft(content, "\ufeff\n\r\t ")
	if !bytes.HasPrefix(trimmed, []byte("---\n")) {
		return nil, ErrNoFrontmatter
	}

	rest := trimmed[len("---\n"):]
	end := bytes.Index(rest, []byte("---\n"))
	if end == -1 {
		return nil, ErrNoFrontmatter
	}

	yamlBytes := rest[:end]
	body := string(rest[end+len("---\n"):])

	var frontmatter map[string]any
	if err := yaml.Unmarshal(yamlBytes, &frontmatter); err != nil {
		return nil, ErrInvalidYAML
	}

	title, ok := frontmatter["title"].(string)
	if !ok || strings.TrimSpace(title) == "" {
		return nil, ErrMissingTitle
	}

	entityType, ok := frontmatter["type"].(string)
	if !ok || strings.TrimSpace(entityType) == "" {
		return nil, ErrMissingType
	}

	tags, err := parseTags(frontmatter["tags"])
	if err != nil {
		return nil, err
	}

	return &Document{
		Frontmatter: frontmatter,
		Title:       title,
		EntityType:  entityType,
		Tags:        tags,
		Body:        body,
	}, nil
}

func parseTags(value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		return []string{v}, nil
	case []any:
		tags := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("tags must be strings")
			}
			if strings.TrimSpace(s) == "" {
				continue
			}
			tags = append(tags, s)
		}
		if len(tags) == 0 {
			return nil, nil
		}
		return tags, nil
	default:
		return nil, fmt.Errorf("tags must be string or list of strings")
	}
}
