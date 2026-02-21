package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"lorecraft/internal/config"
	"lorecraft/internal/parser"
	"lorecraft/internal/store"
)

type Result struct {
	NodesUpserted int
	EdgesUpserted int
	NodesRemoved  int
	FilesSkipped  int
	Errors        []error
}

type Options struct {
	Full bool
}

type processedDoc struct {
	doc   *parser.Document
	layer config.Layer
}

func Run(ctx context.Context, cfg *config.ProjectConfig, schema *config.Schema, db Store, options Options) (*Result, error) {
	if err := db.EnsureSchema(ctx, schema); err != nil {
		return nil, fmt.Errorf("ensure schema: %w", err)
	}

	result := &Result{}
	var processed []processedDoc
	layerFiles := make(map[string][]string)

	for _, layer := range cfg.Layers {
		var existingHashes map[string]string
		if !options.Full {
			var err error
			existingHashes, err = db.GetLayerHashes(ctx, layer.Name)
			if err != nil {
				return nil, fmt.Errorf("get layer hashes for %s: %w", layer.Name, err)
			}
		}

		files, err := walkMarkdownFiles(layer.Paths, cfg.Exclude)
		if err != nil {
			return nil, fmt.Errorf("walking files for layer %s: %w", layer.Name, err)
		}
		layerFiles[layer.Name] = files

		for _, path := range files {
			hash, err := computeHash(path)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("hashing %s: %w", path, err))
				continue
			}
			if !options.Full {
				if existing, ok := existingHashes[path]; ok && existing == hash {
					result.FilesSkipped++
					continue
				}
			}

			doc, err := parser.ParseFile(path)
			if err != nil {
				if err == parser.ErrNoFrontmatter || err == parser.ErrMissingType {
					result.FilesSkipped++
					continue
				}
				result.Errors = append(result.Errors, fmt.Errorf("parsing %s: %w", path, err))
				continue
			}

			if !schema.IsValidEntityType(doc.EntityType) {
				result.FilesSkipped++
				continue
			}

			entityType, _ := schema.EntityTypeByName(doc.EntityType)
			props := filterProperties(doc.Frontmatter, entityType)

			if strings.EqualFold(doc.EntityType, "event") {
				if value, ok := doc.Frontmatter["consequences"]; ok {
					consequences, err := parseConsequences(value)
					if err != nil {
						result.Errors = append(result.Errors, fmt.Errorf("parsing consequences in %s: %w", path, err))
						continue
					}
					payload, err := json.Marshal(consequences)
					if err != nil {
						result.Errors = append(result.Errors, fmt.Errorf("encoding consequences in %s: %w", path, err))
						continue
					}
					if props == nil {
						props = make(map[string]any)
					}
					props["consequences_json"] = string(payload)
				}
			}

			input := store.EntityInput{
				Name:       doc.Title,
				EntityType: doc.EntityType,
				Layer:      layer.Name,
				SourceFile: path,
				SourceHash: hash,
				Properties: props,
				Tags:       doc.Tags,
				Body:       doc.Body,
			}

			if err := db.UpsertEntity(ctx, input); err != nil {
				result.Errors = append(result.Errors, fmt.Errorf("upserting %s: %w", path, err))
				continue
			}
			result.NodesUpserted++
			processed = append(processed, processedDoc{doc: doc, layer: layer})
		}
	}

	for _, item := range processed {
		entityType, _ := schema.EntityTypeByName(item.doc.EntityType)
		for _, mapping := range entityType.FieldMappings {
			if value, ok := item.doc.Frontmatter[mapping.Field]; ok {
				for _, target := range resolveFieldValue(value) {
					if target == "" {
						continue
					}
					targetLayer := item.layer.Name
					if len(item.layer.DependsOn) > 0 {
						layers := append([]string{item.layer.Name}, item.layer.DependsOn...)
						layerName, err := db.FindEntityLayer(ctx, target, layers)
						if err != nil {
							result.Errors = append(result.Errors, fmt.Errorf("finding layer for %s: %w", target, err))
							continue
						}
						if layerName != "" {
							targetLayer = layerName
						}
					}
					if err := db.UpsertRelationship(ctx, item.doc.Title, item.layer.Name, target, targetLayer, mapping.Relationship); err != nil {
						result.Errors = append(result.Errors, fmt.Errorf("upserting relationship for %s: %w", item.doc.Title, err))
						continue
					}
					result.EdgesUpserted++
				}
			}
		}

		if value, ok := item.doc.Frontmatter["related"]; ok {
			for _, target := range resolveFieldValue(value) {
				if target == "" {
					continue
				}
				targetLayer := item.layer.Name
				if len(item.layer.DependsOn) > 0 {
					layers := append([]string{item.layer.Name}, item.layer.DependsOn...)
					layerName, err := db.FindEntityLayer(ctx, target, layers)
					if err != nil {
						result.Errors = append(result.Errors, fmt.Errorf("finding layer for %s: %w", target, err))
						continue
					}
					if layerName != "" {
						targetLayer = layerName
					}
				}
				if err := db.UpsertRelationship(ctx, item.doc.Title, item.layer.Name, target, targetLayer, "RELATED_TO"); err != nil {
					result.Errors = append(result.Errors, fmt.Errorf("upserting related for %s: %w", item.doc.Title, err))
					continue
				}
				result.EdgesUpserted++
			}
		}
	}

	for _, layer := range cfg.Layers {
		deleted, err := db.RemoveStaleNodes(ctx, layer.Name, layerFiles[layer.Name])
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("removing stale nodes for %s: %w", layer.Name, err))
			continue
		}
		result.NodesRemoved += int(deleted)
	}

	return result, nil
}

func walkMarkdownFiles(roots []string, excludes []string) ([]string, error) {
	excluded := make([]string, 0, len(excludes))
	for _, path := range excludes {
		if path == "" {
			continue
		}
		excluded = append(excluded, filepath.Clean(path))
	}

	var files []string
	for _, root := range roots {
		if root == "" {
			continue
		}
		root = filepath.Clean(root)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && isExcluded(path, excluded) {
				return filepath.SkipDir
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
				return nil
			}
			if isExcluded(path, excluded) {
				return nil
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func isExcluded(path string, excludes []string) bool {
	clean := filepath.Clean(path)
	for _, exclude := range excludes {
		if exclude == clean || strings.HasPrefix(clean, exclude+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func computeHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func resolveFieldValue(value any) []string {
	if value == nil {
		return []string{}
	}
	switch v := value.(type) {
	case string:
		return []string{v}
	case []any:
		values := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if ok {
				values = append(values, s)
			}
		}
		return values
	default:
		return []string{}
	}
}

func filterProperties(frontmatter map[string]any, entityType *config.EntityType) map[string]any {
	if frontmatter == nil || entityType == nil {
		return nil
	}

	props := make(map[string]any)
	for key, value := range frontmatter {
		if key == "title" || key == "type" || key == "tags" || key == "related" || key == "consequences" {
			continue
		}
		if isFieldMapping(entityType, key) {
			continue
		}
		if !isProperty(entityType, key) {
			continue
		}
		props[key] = value
	}

	return props
}

func isProperty(entityType *config.EntityType, key string) bool {
	for _, prop := range entityType.Properties {
		if prop.Name == key {
			return true
		}
	}
	return false
}

func isFieldMapping(entityType *config.EntityType, key string) bool {
	for _, mapping := range entityType.FieldMappings {
		if mapping.Field == key {
			return true
		}
	}
	return false
}
