package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
	"stratum/config"
	"stratum/db"
)

// Helper to setup a test directory structure and config files
func setupTestEnv(t *testing.T) (string, func()) {
	// Create temporary directories
	err := os.MkdirAll("config", 0755)
	if err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	err = os.MkdirAll("data/jsonl", 0755)
	if err != nil {
		t.Fatalf("failed to create data dir: %v", err)
	}

	// Create a temporary database file
	tmpFile, err := os.CreateTemp("", "stratum_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	os.Remove(dbPath) // Let DuckDB initialize it

	// Write default YAML config
	cfgContent := fmt.Sprintf(`api:
  keys: ["test-api-key"]
  email: "test@example.com"
filters:
  date_from: "2024-01-01"
  date_to: "2024-12-31"
  doc_types: ["article"]
collection:
  per_page: 50
  concurrent_requests: 2
  max_retries: 3
  retry_delay: 1
llm:
  provider: "ollama"
  model: "llama3"
output:
  jsonl_dir: "data/jsonl"
  db_dir: "%s"
keywords_file: "config/keywords.txt"
topics_file: "config/topics.txt"
anchor_file: "config/anchor.txt"
`, filepath.Dir(dbPath))

	err = os.WriteFile("config/collection.yml", []byte(cfgContent), 0644)
	if err != nil {
		t.Fatalf("failed to write collection.yml: %v", err)
	}

	err = os.WriteFile("config/keywords.txt", []byte("(\"machine learning\")"), 0644)
	if err != nil {
		t.Fatalf("failed to write keywords.txt: %v", err)
	}

	err = os.WriteFile("config/topics.txt", []byte("T10012\nT10245"), 0644)
	if err != nil {
		t.Fatalf("failed to write topics.txt: %v", err)
	}

	err = os.WriteFile("config/anchor.txt", []byte("10.1001/test"), 0644)
	if err != nil {
		t.Fatalf("failed to write anchor.txt: %v", err)
	}

	cleanup := func() {
		os.Remove("config/collection.yml")
		os.Remove("config/keywords.txt")
		os.Remove("config/topics.txt")
		os.Remove("config/anchor.txt")
		os.RemoveAll("config")
		os.RemoveAll("data")
		os.Remove(dbPath)
		os.Remove(dbPath + ".tmp")
		os.Remove(dbPath + ".wal")
	}

	return dbPath, cleanup
}

type redirectTransport struct {
	targetURL     *url.URL
	origTransport http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = t.targetURL.Scheme
	req.URL.Host = t.targetURL.Host
	return t.origTransport.RoundTrip(req)
}

func TestStatusRoute(t *testing.T) {
	dbPath, cleanup := setupTestEnv(t)
	defer cleanup()

	server := NewAPIServer("localhost:8080", dbPath)
	err := server.RegisterRoutes()
	if err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "online" {
		t.Errorf("expected status online, got %q", resp["status"])
	}
}

func TestStatsRoute(t *testing.T) {
	dbPath, cleanup := setupTestEnv(t)
	defer cleanup()

	server := NewAPIServer("localhost:8080", dbPath)
	err := server.RegisterRoutes()
	if err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	// Prepare mock schema and mock data
	err = server.dbMgr.CreateSchema()
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	dbConn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer dbConn.Close()

	_, err = dbConn.Exec(`
		INSERT INTO papers (id, doi, title, publication_year, journal_name, is_oa, cited_by_count)
		VALUES ('p-1', '10.1001/test1', 'Deep Learning', 2024, 'Nature', TRUE, 100)
	`)
	if err != nil {
		t.Fatalf("failed to insert mock paper: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/stats", nil)
	w := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var stats db.DashboardStats
	json.NewDecoder(w.Body).Decode(&stats)

	if stats.TotalPapers != 1 {
		t.Errorf("expected TotalPapers = 1, got %d", stats.TotalPapers)
	}
}

func TestQueryRoute(t *testing.T) {
	dbPath, cleanup := setupTestEnv(t)
	defer cleanup()

	server := NewAPIServer("localhost:8080", dbPath)
	err := server.RegisterRoutes()
	if err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	err = server.dbMgr.CreateSchema()
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	dbConn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open duckdb: %v", err)
	}
	defer dbConn.Close()

	_, err = dbConn.Exec(`
		INSERT INTO papers (id, doi, title, publication_year, journal_name, is_oa)
		VALUES ('p-1', '10.1001/test1', 'Deep Learning', 2024, 'Nature', TRUE)
	`)
	if err != nil {
		t.Fatalf("failed to insert mock paper: %v", err)
	}

	bodyJSON, _ := json.Marshal(map[string]string{
		"query": "SELECT id, title FROM papers WHERE publication_year = 2024",
	})

	req := httptest.NewRequest("POST", "/api/query", bytes.NewBuffer(bodyJSON))
	w := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var rows []map[string]interface{}
	json.NewDecoder(w.Body).Decode(&rows)

	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	} else {
		if rows[0]["id"] != "p-1" || rows[0]["title"] != "Deep Learning" {
			t.Errorf("unexpected row content: %v", rows[0])
		}
	}
}

func TestConfigRoute(t *testing.T) {
	dbPath, cleanup := setupTestEnv(t)
	defer cleanup()

	server := NewAPIServer("localhost:8080", dbPath)
	err := server.RegisterRoutes()
	if err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	// 1. Test GET config
	reqGet := httptest.NewRequest("GET", "/api/config", nil)
	wGet := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Errorf("expected GET status 200, got %d", wGet.Code)
	}

	var respGet map[string]interface{}
	json.NewDecoder(wGet.Body).Decode(&respGet)

	if respGet["keywords"] != "(\"machine learning\")" {
		t.Errorf("expected keywords matching config, got %q", respGet["keywords"])
	}

	// 2. Test POST config
	cfg, err := config.LoadConfig("config/collection.yml")
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	bodyJSON, _ := json.Marshal(map[string]interface{}{
		"config":   cfg,
		"keywords": "(\"neural networks\")",
		"topics":   "T22334",
		"anchors":  "10.1111/test",
	})

	reqPost := httptest.NewRequest("POST", "/api/config", bytes.NewBuffer(bodyJSON))
	wPost := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(wPost, reqPost)

	if wPost.Code != http.StatusOK {
		t.Errorf("expected POST status 200, got %d", wPost.Code)
	}

	// Check if file content updated
	kData, _ := os.ReadFile("config/keywords.txt")
	if string(kData) != "(\"neural networks\")" {
		t.Errorf("expected keywords updated to '(\"neural networks\")', got %q", string(kData))
	}
}

func TestSPAFallback(t *testing.T) {
	dbPath, cleanup := setupTestEnv(t)
	defer cleanup()

	server := NewAPIServer("localhost:8080", dbPath)
	err := server.RegisterRoutes()
	if err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	// Hitting /sql or /docs should serve /dist/index.html (the SPA shell)
	req := httptest.NewRequest("GET", "/sql", nil)
	w := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected fallback status 200, got %d", w.Code)
	}

	bodyStr := w.Body.String()
	if !strings.Contains(bodyStr, "<html") && !strings.Contains(bodyStr, "Stratum") {
		t.Errorf("expected SPA fallback to render HTML document shell, got: %s", bodyStr)
	}
}

func TestPipelineRoutes(t *testing.T) {
	dbPath, cleanup := setupTestEnv(t)
	defer cleanup()

	server := NewAPIServer("localhost:8080", dbPath)
	err := server.RegisterRoutes()
	if err != nil {
		t.Fatalf("RegisterRoutes failed: %v", err)
	}

	// Mock OpenAlex API serving pagination query
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Return empty search page response
		payload := map[string]interface{}{
			"meta": map[string]interface{}{
				"count":       0,
				"next_cursor": "",
			},
			"results": []interface{}{},
		}
		json.NewEncoder(w).Encode(payload)
	}))
	defer ts.Close()

	// Redirect HTTP calls to test server
	mockURL, _ := url.Parse(ts.URL)
	origTransport := http.DefaultTransport
	http.DefaultTransport = &redirectTransport{targetURL: mockURL, origTransport: origTransport}
	defer func() { http.DefaultTransport = origTransport }()

	// Trigger run pipeline
	reqRun := httptest.NewRequest("POST", "/api/run-pipeline", nil)
	wRun := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(wRun, reqRun)

	if wRun.Code != http.StatusOK {
		t.Errorf("expected /api/run-pipeline status 200, got %d", wRun.Code)
	}

	// Give the async pipeline goroutine a brief moment to run and close
	time.Sleep(100 * time.Millisecond)

	// Fetch status
	reqStatus := httptest.NewRequest("GET", "/api/pipeline/status", nil)
	wStatus := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(wStatus, reqStatus)

	if wStatus.Code != http.StatusOK {
		t.Errorf("expected /api/pipeline/status status 200, got %d", wStatus.Code)
	}

	var status PipelineStatus
	json.NewDecoder(wStatus.Body).Decode(&status)

	// In the test setup, OpenAlex returns 0 results, so it should finish very fast and set Syncing to false.
	// Check that logs list is populated.
	if len(status.Logs) == 0 {
		t.Errorf("expected logs captured, got 0 logs")
	}
}
