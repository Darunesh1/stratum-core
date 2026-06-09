package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPServer manages the official Model Context Protocol server state.
type MCPServer struct {
	name    string
	version string
	server  *mcp.Server
}

// NewMCPServer initializes a new MCP server instance.
func NewMCPServer(name, version string) *MCPServer {
	return &MCPServer{name: name, version: version}
}

// ValidateArgs defines the input schema for the validate tool.
type ValidateArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"description=Optional path to the collection.yml config file (default: config/collection.yml)"`
}

// ValidateResult defines the output schema for the validate tool.
type ValidateResult struct {
	Valid  bool     `json:"valid" jsonschema:"description=Indicates if keywords and topics are structurally valid and exist in OpenAlex"`
	Errors []string `json:"errors" jsonschema:"description=List of validation error messages, empty if valid"`
}

// SearchArgs defines the input schema for the search tool.
type SearchArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"description=Optional path to the collection.yml config file (default: config/collection.yml)"`
}

// SearchResult defines the output schema for the search tool.
type SearchResult struct {
	TotalCount int `json:"total_count" jsonschema:"description=The total number of academic papers matching the query parameters in OpenAlex"`
}

// DownloadArgs defines the input schema for the download tool.
type DownloadArgs struct {
	ConfigPath  string `json:"config_path,omitempty" jsonschema:"description=Optional path to the collection.yml config file (default: config/collection.yml)"`
	OutputJSONL string `json:"output_jsonl,omitempty" jsonschema:"description=Optional path to write the downloaded JSONL file (defaults to config output location)"`
}

// DownloadResult defines the output schema for the download tool.
type DownloadResult struct {
	Status  string `json:"status" jsonschema:"description=Status message indicating success or failure"`
	Message string `json:"message" jsonschema:"description=Detail message outlining details of downloaded papers"`
}

// ConvertDBArgs defines the input schema for the convert_db tool.
type ConvertDBArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"description=Optional path to the collection.yml config file (default: config/collection.yml)"`
	JSONLPath  string `json:"jsonl_path,omitempty" jsonschema:"description=Optional path to the input JSONL file (defaults to latest downloaded)"`
}

// ConvertDBResult defines the output schema for the convert_db tool.
type ConvertDBResult struct {
	Status        string `json:"status" jsonschema:"description=Status message indicating success or failure"`
	PapersLoaded  int    `json:"papers_loaded" jsonschema:"description=Number of papers successfully loaded into DuckDB"`
	AuthorsLoaded int    `json:"authors_loaded" jsonschema:"description=Number of unique authors loaded"`
	InstsLoaded   int    `json:"institutions_loaded" jsonschema:"description=Number of unique institutions loaded"`
	Errors        int    `json:"errors" jsonschema:"description=Number of row errors encountered during ingestion"`
}

// ImputeArgs defines the input schema for the impute tool.
type ImputeArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"description=Optional path to the collection.yml config file (default: config/collection.yml)"`
	Pipeline   string `json:"pipeline,omitempty" jsonschema:"description=Pipeline stage to execute: crossref, llm, pdf, or all (default: all)"`
	Limit      int    `json:"limit,omitempty" jsonschema:"description=Optional limit for the number of PDF files to download and process"`
}

// ImputeResult defines the output schema for the impute tool.
type ImputeResult struct {
	Status  string `json:"status" jsonschema:"description=Status message indicating success or failure"`
	Message string `json:"message" jsonschema:"description=Detailed summary of actions taken and records updated"`
}

// RegisterTools registers all available collection, database, and imputation pipeline tools.
func (s *MCPServer) RegisterTools() error {
	// TODO: Use mcp.AddTool to register validate, search, download, convert_db, and impute handlers.
	return nil
}

// Start runs the MCP server on the stdio transport interface.
func (s *MCPServer) Start(ctx context.Context) error {
	// TODO: Create mcp.Server instance, register tools, bind to StdioTransport, and run.
	return nil
}

// handleValidate validates the keywords syntax and checks if configured topics exist in OpenAlex.
func (s *MCPServer) handleValidate(ctx context.Context, req *mcp.CallToolRequest, args ValidateArgs) (*mcp.CallToolResult, any, error) {
	// TODO: Implementation details
	return nil, nil, nil
}

// handleSearch queries OpenAlex to return the count of matching papers.
func (s *MCPServer) handleSearch(ctx context.Context, req *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, any, error) {
	// TODO: Implementation details
	return nil, nil, nil
}

// handleDownload downloads papers matching query filters concurrently and saves them to JSONL.
func (s *MCPServer) handleDownload(ctx context.Context, req *mcp.CallToolRequest, args DownloadArgs) (*mcp.CallToolResult, any, error) {
	// TODO: Implementation details
	return nil, nil, nil
}

// handleConvertDB imports downloaded JSONL paper records into DuckDB.
func (s *MCPServer) handleConvertDB(ctx context.Context, req *mcp.CallToolRequest, args ConvertDBArgs) (*mcp.CallToolResult, any, error) {
	// TODO: Implementation details
	return nil, nil, nil
}

// handleImpute imputes missing institution and country metadata using Crossref, LLMs, and PDF text extraction.
func (s *MCPServer) handleImpute(ctx context.Context, req *mcp.CallToolRequest, args ImputeArgs) (*mcp.CallToolResult, any, error) {
	// TODO: Implementation details
	return nil, nil, nil
}
