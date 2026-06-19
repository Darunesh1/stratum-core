---
sidebar_position: 5
---

# Model Context Protocol (MCP)

Stratum implements the Model Context Protocol (MCP) to allow AI assistants (such as Claude Desktop or Gemini) to run database queries, manage configuration, download records, and trigger imputation pipelines directly on your local system.

---

## 1. Running the MCP Server

The MCP server runs over **stdio transport**. There are two ways to execute the MCP server:

### A. Integrated (Via Stratum Core)
The main Go server can start in MCP-only mode by passing the `-mcp` flag:
```bash
./stratum -mcp
```

### B. Standalone (Via `stratum-mcp` or NPM)
If you only need the MCP capabilities without the dashboard interface or SQLite storage, you can run the standalone Go binary:
```bash
# Run standalone Go binary
./stratum-mcp

# Or execute via npx
npx stratum-mcp
```

---

## 2. Configuration for Claude Desktop

To integrate Stratum MCP with Claude Desktop, add the server configuration to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "stratum-mcp": {
      "command": "npx",
      "args": ["-y", "stratum-mcp"],
      "env": {
        "GEMINI_API_KEY": "YOUR_GEMINI_API_KEY"
      }
    }
  }
}
```

*Note: Make sure to define the `GEMINI_API_KEY` environment variable if you plan to execute LLM-based metadata imputation.*

---

## 3. Registered Tools

The MCP server exposes the following tools:

### 1. `validate`
Validates the OpenAlex search keywords syntax and checks if all configured topics exist.

- **Parameters:**
  - `ConfigPath` (string, optional): Absolute path to the config file (defaults to `data/db/config.db`).
- **Response Format:**
  ```json
  {
    "valid": true,
    "errors": []
  }
  ```

### 2. `search`
Queries the OpenAlex API to return the total count of academic papers matching the current filters.

- **Parameters:**
  - `ConfigPath` (string, optional): Absolute path to the config file.
- **Response Format:**
  ```json
  {
    "count": 2780
  }
  ```

### 3. `download`
Downloads papers matching the configuration concurrently, saving records to a local JSONL file.

- **Parameters:**
  - `ConfigPath` (string, optional): Absolute path to the config file.
  - `OutputPath` (string, required): Destination path for the downloaded `.jsonl` file.
- **Response Format:**
  ```json
  {
    "saved_path": "data/downloads/papers.jsonl",
    "total_records": 2780
  }
  ```

### 4. `convert_db`
Parses the downloaded JSONL paper records and inserts them into DuckDB. It initializes the tables and database sequences.

- **Parameters:**
  - `ConfigPath` (string, optional): Absolute path to the config file.
  - `JSONLPath` (string, required): Path to the raw JSONL file.
  - `DBPath` (string, required): Destination path for the `.db` DuckDB file.
- **Response Format:**
  ```json
  {
    "papers_loaded": 2780,
    "authors_loaded": 6842,
    "institutions_loaded": 219
  }
  ```

### 5. `impute`
Scans DuckDB for missing author countries and affiliations, and triggers the Crossref/LLM/PDF imputation loop to enrich records.

- **Parameters:**
  - `ConfigPath` (string, optional): Absolute path to the config file.
  - `DBPath` (string, required): Path to the DuckDB file.
  - `UseLLM` (boolean, optional): Enable Gemini LLM string classification.
  - `PDFDir` (string, optional): Directory containing PDF manuscripts for text parsing.
- **Response Format:**
  ```json
  {
    "status": "success",
    "message": "Imputation pipeline complete. Updated 102 country codes and 45 institutions."
  }
  ```
