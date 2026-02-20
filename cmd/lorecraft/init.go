package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	var projectName string
	var template string
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold a new lorecraft project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(projectName) == "" {
				return fmt.Errorf("--name is required")
			}
			return runInit(projectName, template)
		},
	}
	cmd.Flags().StringVar(&projectName, "name", "", "Project name")
	cmd.Flags().StringVar(&template, "template", "fantasy-rpg", "Schema template name")
	return cmd
}

func runInit(projectName, template string) error {
	configPath := "lorecraft.yaml"
	schemaPath := "schema.yaml"
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("%s already exists", configPath)
	}
	if _, err := os.Stat(schemaPath); err == nil {
		return fmt.Errorf("%s already exists", schemaPath)
	}

	templatePath := filepath.Join("schemas", template+".yaml")
	contents, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", template, err)
	}

	configContents := fmt.Sprintf("project: %s\nversion: 1\n\nneo4j:\n  uri: bolt://localhost:7687\n  username: neo4j\n  password: changeme\n  database: neo4j\n\nlayers:\n  - name: setting\n    paths:\n      - ./lore/\n    canonical: true\n\nexclude:\n  - ./assets/\n", projectName)
	if err := os.WriteFile(configPath, []byte(configContents), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", configPath, err)
	}
	if err := os.WriteFile(schemaPath, contents, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", schemaPath, err)
	}

	return nil
}
