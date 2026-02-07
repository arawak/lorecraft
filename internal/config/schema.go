package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Schema struct {
	Version           int                `yaml:"version"`
	EntityTypes       []EntityType       `yaml:"entity_types"`
	RelationshipTypes []RelationshipType `yaml:"relationship_types"`

	entityIndex map[string]*EntityType
	relIndex    map[string]*RelationshipType
}

type EntityType struct {
	Name          string         `yaml:"name"`
	Properties    []Property     `yaml:"properties"`
	FieldMappings []FieldMapping `yaml:"field_mappings"`
}

type Property struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Values   []string `yaml:"values"`
	Default  string   `yaml:"default"`
	Required bool     `yaml:"required"`
}

type FieldMapping struct {
	Field        string   `yaml:"field"`
	Relationship string   `yaml:"relationship"`
	TargetType   []string `yaml:"target_type"`
}

type RelationshipType struct {
	Name      string `yaml:"name"`
	Inverse   string `yaml:"inverse"`
	Symmetric bool   `yaml:"symmetric"`
}

func LoadSchema(path string) (*Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}

	var schema Schema
	if err := yaml.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}

	if err := validateSchema(&schema); err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}

	schema.entityIndex = make(map[string]*EntityType)
	for i := range schema.EntityTypes {
		entity := &schema.EntityTypes[i]
		schema.entityIndex[strings.ToLower(entity.Name)] = entity
	}

	schema.relIndex = make(map[string]*RelationshipType)
	for i := range schema.RelationshipTypes {
		rel := &schema.RelationshipTypes[i]
		schema.relIndex[strings.ToLower(rel.Name)] = rel
	}

	return &schema, nil
}

func validateSchema(s *Schema) error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported version: %d", s.Version)
	}
	if len(s.EntityTypes) == 0 {
		return fmt.Errorf("at least one entity type is required")
	}

	entityNames := make(map[string]struct{})
	for i, entity := range s.EntityTypes {
		if strings.TrimSpace(entity.Name) == "" {
			return fmt.Errorf("entity type %d name is required", i)
		}
		key := strings.ToLower(entity.Name)
		if _, exists := entityNames[key]; exists {
			return fmt.Errorf("duplicate entity type name: %s", entity.Name)
		}
		entityNames[key] = struct{}{}

		propNames := make(map[string]struct{})
		for _, prop := range entity.Properties {
			name := strings.ToLower(strings.TrimSpace(prop.Name))
			if name == "" {
				return fmt.Errorf("entity type %s has property with empty name", entity.Name)
			}
			if _, exists := propNames[name]; exists {
				return fmt.Errorf("entity type %s has duplicate property: %s", entity.Name, prop.Name)
			}
			propNames[name] = struct{}{}
			if strings.EqualFold(prop.Type, "enum") && len(prop.Values) == 0 {
				return fmt.Errorf("entity type %s property %s enum has no values", entity.Name, prop.Name)
			}
		}
	}

	relNames := make(map[string]struct{})
	for i, rel := range s.RelationshipTypes {
		if strings.TrimSpace(rel.Name) == "" {
			return fmt.Errorf("relationship type %d name is required", i)
		}
		key := strings.ToLower(rel.Name)
		if _, exists := relNames[key]; exists {
			return fmt.Errorf("duplicate relationship type name: %s", rel.Name)
		}
		relNames[key] = struct{}{}
	}

	for _, entity := range s.EntityTypes {
		for _, mapping := range entity.FieldMappings {
			if strings.TrimSpace(mapping.Field) == "" {
				return fmt.Errorf("entity type %s has field mapping with empty field", entity.Name)
			}
			if strings.TrimSpace(mapping.Relationship) == "" {
				return fmt.Errorf("entity type %s has field mapping with empty relationship", entity.Name)
			}
			if _, ok := relNames[strings.ToLower(mapping.Relationship)]; !ok {
				return fmt.Errorf("entity type %s field mapping references unknown relationship: %s", entity.Name, mapping.Relationship)
			}
		}
	}

	return nil
}

func (s *Schema) EntityTypeByName(name string) (*EntityType, bool) {
	if s == nil {
		return nil, false
	}
	entity, ok := s.entityIndex[strings.ToLower(name)]
	return entity, ok
}

func (s *Schema) RelationshipTypeByName(name string) (*RelationshipType, bool) {
	if s == nil {
		return nil, false
	}
	rel, ok := s.relIndex[strings.ToLower(name)]
	return rel, ok
}

func (s *Schema) IsValidEntityType(name string) bool {
	_, ok := s.EntityTypeByName(name)
	return ok
}

func (s *Schema) IsValidRelationshipType(name string) bool {
	_, ok := s.RelationshipTypeByName(name)
	return ok
}

func (s *Schema) NodeLabel(entityType string) string {
	return strings.ToUpper(entityType)
}
