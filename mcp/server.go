package mcp

import (
	"context"
)

// MCPServer manages the official Model Context Protocol server state.
type MCPServer struct {
	name    string
	version string
}

// NewMCPServer initializes a new MCP server instances.
func NewMCPServer(name, version string) *MCPServer {
	return &MCPServer{name: name, version: version}
}

// RegisterTools registers all available collection, database, and imputation pipeline tools.
func (s *MCPServer) RegisterTools() error {
	// TODO: Use mcp.AddTool to expose validate, search, download, convert-db, and impute.
	return nil
}

// Start runs the MCP server on the stdio transport interface.
func (s *MCPServer) Start(ctx context.Context) error {
	// TODO: Bind server to mcp.StdioTransport{} and listen for commands.
	return nil
}
