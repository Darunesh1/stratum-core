# MCP Tools Catalog

This guide lists all registered tools in the Stratum MCP server.

*   **`configure_project`**: Read/Write project configurations (`config.db` / SQLite).
*   **`upload_file`**: Copies a local workspace file (CSV/Excel) to the project uploads directory.
*   **`extract_query_and_anchors`**: Runs TF-IDF keyword extraction and DOI-based anchor extraction.
*   **`validate`**: Validates boolean keyword syntax and topic existence on OpenAlex.
*   **`search`**: Retrieves the total count of academic papers and verifies anchor coverage.
*   *   **`get_sample`**: Fetches a random validation sample of papers matching current search rules.
*   **`get_topics`**: Fetches topic distribution matching the filters.
*   **`download`**: Downloads matching paper metadata from OpenAlex to JSONL.
*   **`convert_db`**: Ingests JSONL records into DuckDB.
*   **`impute`**: Triggers metadata enrichment (Crossref, LLMs, PDF text).
