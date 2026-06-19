---
sidebar_position: 4
---

# API Reference

The Stratum backend exposes a REST HTTP API for managing projects, configuring data sources, executing database queries, and checking pipeline runs. All endpoints are prefixed with `/api/`.

---

## 1. System Status

### `GET /api/status`
Returns the operational health status of the Go server.

**Response (200 OK):**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 1284
}
```

---

## 2. Ingest & Processing Pipelines

### `POST /api/run-pipeline`
Triggers a background ingestion or metadata imputation pipeline execution.

**Request Body:**
```json
{
  "action": "ingest" // Options: "ingest", "impute"
}
```

**Response (200 OK):**
```json
{
  "status": "started",
  "pipeline_id": "pip-902318",
  "message": "Ingestion pipeline initialized in the background."
}
```

---

### `GET /api/pipeline/status`
Polls the active or most recent pipeline execution state, returning progress statistics.

**Response (200 OK):**
```json
{
  "pipeline_id": "pip-902318",
  "status": "running", // Options: "idle", "running", "completed", "failed"
  "action": "ingest",
  "progress_percent": 45,
  "processed_items": 1250,
  "total_items": 2778,
  "error_count": 0
}
```

---

## 3. Database Statistics & Querying

### `GET /api/stats`
Retrieves summary metrics from the loaded DuckDB analytical database.

**Response (200 OK):**
```json
{
  "total_papers": 15004,
  "total_authors": 38102,
  "total_institutions": 1492,
  "missing_country_count": 240,
  "imputed_country_count": 105,
  "international_collaboration_rate": 0.28
}
```

---

### `POST /api/query`
Executes an analytical SQL statement against the DuckDB database. Only read-only queries (e.g. `SELECT`) are permitted.

**Request Body:**
```json
{
  "sql": "SELECT publication_year, COUNT(*) as paper_count FROM papers GROUP BY publication_year ORDER BY publication_year DESC"
}
```

**Response (200 OK):**
```json
{
  "columns": ["publication_year", "paper_count"],
  "rows": [
    [2026, 451],
    [2025, 2304],
    [2024, 2190]
  ],
  "execution_time_ms": 12
}
```

**Response (400 Bad Request):**
```json
{
  "error": "Access Denied: only read-only statements are allowed."
}
```

---

### `POST /api/query/validate`
Validates the syntax and safety of a SQL query before execution.

**Request Body:**
```json
{
  "sql": "SELECT * FROM papers"
}
```

**Response (200 OK):**
```json
{
  "valid": true,
  "error": ""
}
```

---

## 4. Configuration

### `GET /api/config`
Retrieves the current search keywords, topics, and API keys.

**Response (200 OK):**
```json
{
  "keywords": "quantum computing AND cryptography",
  "topics": ["T10123", "T10901"],
  "api": {
    "email": "researcher@university.edu",
    "concurrent_requests": 4
  }
}
```

---

### `POST /api/config`
Updates SQLite configurations.

**Request Body:**
```json
{
  "keywords": "quantum computing AND machine learning",
  "topics": ["T10123"],
  "api": {
    "email": "researcher@university.edu",
    "concurrent_requests": 6
  }
}
```

**Response (200 OK):**
```json
{
  "status": "updated",
  "message": "Configuration successfully saved."
}
```

---

## 5. Metadata Upload & TF-IDF

### `POST /api/upload`
Uploads a local metadata file (e.g., CSV, JSONL) to be parsed and loaded.

**Request:** Multipart form upload containing a file field.

**Response (200 OK):**
```json
{
  "status": "uploaded",
  "filename": "metadata_export.jsonl",
  "records_count": 580
}
```

---

### `POST /api/tfidf`
Runs term frequency-inverse document frequency keyword extraction on paper abstracts.

**Request Body:**
```json
{
  "min_document_frequency": 5,
  "max_n_grams": 2
}
```

**Response (200 OK):**
```json
{
  "keywords": [
    {"term": "quantum cryptography", "score": 0.892},
    {"term": "key distribution", "score": 0.764},
    {"term": "entanglement", "score": 0.541}
  ]
}
```

---

## 6. Projects Management

### `GET /api/projects`
Lists all managed workspaces.

**Response (200 OK):**
```json
[
  {
    "id": "proj-quantum-computing",
    "name": "Quantum Computing Research",
    "created_at": "2026-06-10T08:00:00Z"
  }
]
```

---

### `POST /api/projects/create`
Creates a new isolated workspace.

**Request Body:**
```json
{
  "name": "Carbon Sequestration Ingestion"
}
```

**Response (200 OK):**
```json
{
  "id": "proj-carbon-sequestration",
  "name": "Carbon Sequestration Ingestion",
  "status": "created"
}
```
