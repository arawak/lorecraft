package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
)

func querySearchCmd() *cobra.Command {
	var entityType string
	var layer string
	cmd := &cobra.Command{
		Use:   "search <text>",
		Short: "Search the database using full-text search",
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

	db, err := openDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	results, err := db.Search(ctx, query, layer, entityType)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stdout, "No matches found.")
		return nil
	}

	for _, result := range results {
		fmt.Fprintf(os.Stdout, "%s (%s) [%s] score=%.2f\n", result.Name, result.EntityType, result.Layer, result.Score)
		if result.Snippet != "" {
			fmt.Fprintf(os.Stdout, "  %s\n", result.Snippet)
		}
	}
	return nil
}
