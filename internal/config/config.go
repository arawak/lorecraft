package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProjectConfig struct {
	Project string      `yaml:"project"`
	Version int         `yaml:"version"`
	Neo4j   Neo4jConfig `yaml:"neo4j"`
	Layers  []Layer     `yaml:"layers"`
	Exclude []string    `yaml:"exclude"`
}

type Neo4jConfig struct {
	URI      string `yaml:"uri"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type Layer struct {
	Name      string   `yaml:"name"`
	Paths     []string `yaml:"paths"`
	Canonical bool     `yaml:"canonical"`
	DependsOn []string `yaml:"depends_on"`
}

func LoadProjectConfig(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	if err := validateProjectConfig(&cfg); err != nil {
		return nil, fmt.Errorf("loading project config: %w", err)
	}

	return &cfg, nil
}

func validateProjectConfig(cfg *ProjectConfig) error {
	if strings.TrimSpace(cfg.Project) == "" {
		return fmt.Errorf("project name is required")
	}
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported version: %d", cfg.Version)
	}
	if strings.TrimSpace(cfg.Neo4j.URI) == "" {
		return fmt.Errorf("neo4j uri is required")
	}
	if len(cfg.Layers) == 0 {
		return fmt.Errorf("at least one layer is required")
	}

	seen := make(map[string]struct{})
	layersByName := make(map[string]Layer)
	for i, layer := range cfg.Layers {
		if strings.TrimSpace(layer.Name) == "" {
			return fmt.Errorf("layer %d name is required", i)
		}
		if len(layer.Paths) == 0 {
			return fmt.Errorf("layer %d paths are required", i)
		}
		key := strings.ToLower(layer.Name)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate layer name: %s", layer.Name)
		}
		seen[key] = struct{}{}
		layersByName[key] = layer
	}

	for _, layer := range cfg.Layers {
		for _, dep := range layer.DependsOn {
			depName := strings.TrimSpace(dep)
			if depName == "" {
				continue
			}
			key := strings.ToLower(depName)
			depLayer, ok := layersByName[key]
			if !ok {
				return fmt.Errorf("layer %s depends_on unknown layer %s", layer.Name, depName)
			}
			if layer.Canonical && !depLayer.Canonical {
				return fmt.Errorf("canonical layer %s cannot depend on non-canonical layer %s", layer.Name, depName)
			}
		}
	}

	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(name string) error
	visit = func(name string) error {
		if visiting[name] {
			return fmt.Errorf("dependency cycle detected at layer %s", layersByName[name].Name)
		}
		if visited[name] {
			return nil
		}
		visiting[name] = true
		layer := layersByName[name]
		for _, dep := range layer.DependsOn {
			depName := strings.TrimSpace(dep)
			if depName == "" {
				continue
			}
			if err := visit(strings.ToLower(depName)); err != nil {
				return err
			}
		}
		visiting[name] = false
		visited[name] = true
		return nil
	}

	for name := range layersByName {
		if err := visit(name); err != nil {
			return err
		}
	}

	return nil
}
