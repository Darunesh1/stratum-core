# Stratum Core

Stratum is a high-performance bibliographic metadata retrieval, storage, and enrichment platform. This monorepo houses the Go backend server, SQLite state management, DuckDB analytics layer, and the React frontend dashboard.

---

## Getting Started

### 1. Start the Server
To compile and start the Stratum HTTP API server and React dashboard interface:

```bash
go run main.go
```

By default, this launches the server on `http://localhost:8080`.

### 2. Available Flags
You can customize the server using flags:

```bash
# Change the port (default is 8080)
go run main.go -port 9090

# Specify a custom DuckDB database path (default is data/db/papers.db)
go run main.go -db path/to/my_database.db
```

---

## Project Structure

- **`main.go`**: Entry point that spins up the web and API dashboard.
- **`docs/`**: Central developer documentation folder containing architectural guides, API references, and schemas.
- **`website/`**: Docusaurus static site generator configuration to publish the docs folder online.
- **`web/`**: Single Page Application (SPA) dashboard built using React and Vite.
- **`api/`**: Go HTTP router registering metric calculations, project operations, and pipeline runs.
- **`db/`**: Go wrappers interface to DuckDB.
- **`impute/`**: Imputation pipelines resolving missing metadata via Crossref, LLMs, and local PDF files.

---

## Standalone MCP Server
If you want to use the Model Context Protocol (MCP) server stdio command-line tool directly with AI agents (such as Claude Desktop or Cursor) without the web dashboard or SQLite layers, please visit the dedicated standalone repository: [Stratum MCP](https://github.com/Darunesh1/stratum-mcp).