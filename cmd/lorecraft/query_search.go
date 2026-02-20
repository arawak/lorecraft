package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

func querySearchCmd() *cobra.Command {
	var entityType string
	var layer string
	cmd := &cobra.Command{
		Use:   "search <text>",
		Short: "Search the graph using full-text index",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			return runQuerySearch(cmd, query, entityType, layer)
		},
	}
	cmd.Flags().StringVar(&entityType, "type", "", "Entity type to filter")
	cmd.Flags().StringVar(&layer, "layer", "", "Layer to filter")
	return cmd
}

func runQuerySearch(cmd *cobra.Command, query, entityType, layer string) error {
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

	results, err := client.Search(ctx, query, layer, entityType)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stdout, "No matches found.")
		return nil
	}

	for _, result := range results {
		fmt.Fprintf(os.Stdout, "%s (%s) [%s] score=%.2f\n", result.Name, result.EntityType, result.Layer, result.Score)
	}
	return nil
}
