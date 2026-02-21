package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"lorecraft/internal/config"
	"lorecraft/internal/store"
)

type Server struct {
	schema *config.Schema
	db     store.Store
	mcp    *sdk.Server
}

func NewServer(schema *config.Schema, db store.Store) *Server {
	s := &Server{
		schema: schema,
		db:     db,
		mcp: sdk.NewServer(&sdk.Implementation{
			Name:    "lorecraft",
			Version: "0.1.0",
		}, nil),
	}
	s.registerTools()
	return s
}

func (s *Server) Run(ctx context.Context, transport sdk.Transport) error {
	return s.mcp.Run(ctx, transport)
}
