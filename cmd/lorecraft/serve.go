package main

import (
	"context"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
	"lorecraft/internal/mcp"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func serveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server over stdio",
		RunE:  runServe,
	}
	return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadProjectConfig("lorecraft.yaml")
	if err != nil {
		return err
	}

	schema, err := config.LoadSchema("schema.yaml")
	if err != nil {
		return err
	}

	client, err := graph.NewClient(ctx, cfg.Neo4j.URI, cfg.Neo4j.Username, cfg.Neo4j.Password, cfg.Neo4j.Database)
	if err != nil {
		return err
	}
	defer client.Close(ctx)

	server := mcp.NewServer(schema, client)
	return server.Run(ctx, &sdk.StdioTransport{})
}
