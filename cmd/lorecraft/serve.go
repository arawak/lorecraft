package main

import (
	"context"

	"github.com/spf13/cobra"

	"lorecraft/internal/config"
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

	db, err := openDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close(ctx)

	server := mcp.NewServer(schema, db)
	return server.Run(ctx, &sdk.StdioTransport{})
}
