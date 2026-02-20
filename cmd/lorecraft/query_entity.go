package main

import (
	"context"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

func queryEntityCmd() *cobra.Command {
	var entityType string
	cmd := &cobra.Command{
		Use:   "entity <name>",
		Short: "Display an entity and its properties",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runQueryEntity(cmd, name, entityType)
		},
	}
	cmd.Flags().StringVar(&entityType, "type", "", "Entity type to disambiguate")
	return cmd
}

func runQueryEntity(cmd *cobra.Command, name, entityType string) error {
	ctx := context.Background()

	cfg, err := config.LoadProjectConfig("lorecraft.yaml")
	if err != nil {
		return err
	}

	client, err := graph.NewClient(ctx, cfg.Neo4j.URI, cfg.Neo4j.Username, cfg.Neo4j.Password, cfg.Neo4j.Database)
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	entity, err := client.GetEntity(ctx, name, entityType)
	if err != nil {
		return err
	}
	if entity == nil {
		fmt.Fprintf(os.Stdout, "No entity found for %q.\n", name)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Name: %s\n", entity.Name)
	fmt.Fprintf(os.Stdout, "Type: %s\n", entity.EntityType)
	fmt.Fprintf(os.Stdout, "Layer: %s\n", entity.Layer)
	if len(entity.Tags) > 0 {
		fmt.Fprintf(os.Stdout, "Tags: %s\n", joinValues(entity.Tags))
	}
	if entity.SourceFile != "" {
		fmt.Fprintf(os.Stdout, "Source: %s\n", entity.SourceFile)
	}

	if len(entity.Properties) == 0 {
		return nil
	}

	keys := make([]string, 0, len(entity.Properties))
	for key := range entity.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	fmt.Fprintln(os.Stdout, "Properties:")
	for _, key := range keys {
		fmt.Fprintf(os.Stdout, "  %s: %v\n", key, entity.Properties[key])
	}
	return nil
}
