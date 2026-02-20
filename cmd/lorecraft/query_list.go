package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

func queryListCmd() *cobra.Command {
	var entityType string
	var layer string
	var tag string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List entities in the graph",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runQueryList(cmd, entityType, layer, tag)
		},
	}
	cmd.Flags().StringVar(&entityType, "type", "", "Entity type to filter")
	cmd.Flags().StringVar(&layer, "layer", "", "Layer to filter")
	cmd.Flags().StringVar(&tag, "tag", "", "Tag to filter")
	return cmd
}

func runQueryList(cmd *cobra.Command, entityType, layer, tag string) error {
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

	entities, err := client.ListEntities(ctx, entityType, layer, tag)
	if err != nil {
		return err
	}
	if len(entities) == 0 {
		fmt.Fprintln(os.Stdout, "No entities found.")
		return nil
	}

	for _, entity := range entities {
		fmt.Fprintf(os.Stdout, "%s (%s) [%s]\n", entity.Name, entity.EntityType, entity.Layer)
	}
	return nil
}
