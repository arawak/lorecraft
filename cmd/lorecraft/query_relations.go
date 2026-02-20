package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

func queryRelationsCmd() *cobra.Command {
	var relType string
	var direction string
	var depth int
	cmd := &cobra.Command{
		Use:   "relations <name>",
		Short: "Display relationships for an entity",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			return runQueryRelations(cmd, name, relType, direction, depth)
		},
	}
	cmd.Flags().StringVar(&relType, "type", "", "Relationship type to filter")
	cmd.Flags().StringVar(&direction, "direction", "both", "Direction: outgoing, incoming, or both")
	cmd.Flags().IntVar(&depth, "depth", 1, "Traversal depth (1-5)")
	return cmd
}

func runQueryRelations(cmd *cobra.Command, name, relType, direction string, depth int) error {
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

	rels, err := client.GetRelationships(ctx, name, relType, direction, depth)
	if err != nil {
		return err
	}
	if len(rels) == 0 {
		fmt.Fprintf(os.Stdout, "No relationships found for %q.\n", name)
		return nil
	}

	for _, rel := range rels {
		fmt.Fprintf(os.Stdout, "[%d] %s (%s) -%s-> %s (%s) [%s]\n",
			rel.Depth,
			rel.From.Name,
			rel.From.EntityType,
			rel.Type,
			rel.To.Name,
			rel.To.EntityType,
			rel.Direction,
		)
	}
	return nil
}
