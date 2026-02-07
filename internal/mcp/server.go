package mcp

import (
	"context"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"lorecraft/internal/config"
	"lorecraft/internal/graph"
)

type GraphQuerier interface {
	GetEntity(ctx context.Context, name, entityType string) (*graph.Entity, error)
	GetRelationships(ctx context.Context, name, relType, direction string, depth int) ([]graph.Relationship, error)
	ListEntities(ctx context.Context, entityType, layer, tag string) ([]graph.EntitySummary, error)
	Search(ctx context.Context, query, layer, entityType string) ([]graph.SearchResult, error)
}

type Server struct {
	schema *config.Schema
	graph  GraphQuerier
	mcp    *sdk.Server
}

func NewServer(schema *config.Schema, graph GraphQuerier) *Server {
	s := &Server{
		schema: schema,
		graph:  graph,
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
