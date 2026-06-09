package api

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"stratum/config"
	"stratum/db"
	"stratum/openalex"
)

//go:embed dist/*
var frontendFS embed.FS

// PipelineStatus tracks the state and log output of the active ingestion pipeline.
type PipelineStatus struct {
	Syncing  bool     `json:"syncing"`
	Progress int      `json:"progress"`
	Logs     []string `json:"logs"`
}

// APIServer manages HTTP routes and coordinates tasks for the web client interface.
type APIServer struct {
	addr   string
	dbPath string
	dbMgr  *db.DBManager
	server *http.Server
	status PipelineStatus
	mu     sync.Mutex
}

// NewAPIServer creates a new API server on the specified address.
func NewAPIServer(addr string, dbPath string) *APIServer {
	return &APIServer{
		addr:   addr,
		dbPath: dbPath,
		status: PipelineStatus{
			Syncing:  false,
			Progress: 0,
			Logs:     []string{},
		},
	}
}

// RegisterRoutes registers endpoints for dashboard metrics, query execution, files, and config.
func (s *APIServer) RegisterRoutes() error {
	// Initialize DB manager if not already done
	if s.dbMgr == nil {
		mgr, err := db.NewDBManager(s.dbPath)
		if err != nil {
			return err
		}
		s.dbMgr = mgr
	}

	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/run-pipeline", s.handleRunPipeline)
	mux.HandleFunc("/api/pipeline/status", s.handlePipelineStatus)

	// Get sub-filesystem for frontend assets
	subFS, err := fs.Sub(frontendFS, "dist")
	if err != nil {
		return err
	}

	// Serve embedded static files with client-side SPA routing fallback
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "API endpoint not found"}`))
			return
		}

		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		filePath := strings.TrimPrefix(path, "/")

		// Check if file exists in the embedded directory
		f, err := subFS.Open(filePath)
		if err != nil {
			// Fallback to index.html for client-side routing
			indexData, err := fs.ReadFile(subFS, "index.html")
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(indexData)
			return
		}
		f.Close()

		http.FileServer(http.FS(subFS)).ServeHTTP(w, r)
	})

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	return nil
}

// Start starts the HTTP server asynchronously.
func (s *APIServer) Start() error {
	if s.server == nil {
		return fmt.Errorf("server not registered, call RegisterRoutes first")
	}
	go func() {
		s.server.ListenAndServe()
	}()
	return nil
}

// Stop gracefully shuts down the server.
func (s *APIServer) Stop(ctx context.Context) error {
	if s.dbMgr != nil {
		s.dbMgr.Close()
	}
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// addLog helper pushes a message onto the pipeline log stack
func (s *APIServer) addLog(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	timestamp := time.Now().Format("15:04:05")
	s.status.Logs = append(s.status.Logs, fmt.Sprintf("[%s] %s", timestamp, msg))
}

// updateProgress helper sets current pipeline progress
func (s *APIServer) updateProgress(p int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Progress = p
	if s.status.Progress > 95 {
		s.status.Progress = 95
	}
	timestamp := time.Now().Format("15:04:05")
	s.status.Logs = append(s.status.Logs, fmt.Sprintf("[%s] [INFO] Download progress: %d papers fetched.", timestamp, p))
}

// updateStatus helper modifies run flags and logs completion
func (s *APIServer) updateStatus(syncing bool, progress int, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Syncing = syncing
	s.status.Progress = progress
	timestamp := time.Now().Format("15:04:05")
	s.status.Logs = append(s.status.Logs, fmt.Sprintf("[%s] %s", timestamp, msg))
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "online"}`))
}

func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.dbMgr == nil {
		http.Error(w, "Database manager not initialized", http.StatusInternalServerError)
		return
	}
	stats, err := s.dbMgr.GetDashboardStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats)
}

func (s *APIServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if s.dbMgr == nil {
		http.Error(w, "Database manager not initialized", http.StatusInternalServerError)
		return
	}

	results, err := s.dbMgr.RunQuery(body.Query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (s *APIServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		cfg, err := config.LoadConfig("config/collection.yml")
		if err != nil {
			http.Error(w, "Failed to load config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		keywords, _ := config.GetKeywords(cfg.Keywords)
		topicsData, _ := os.ReadFile(cfg.Topics)
		anchorsData, _ := os.ReadFile(cfg.Anchors)

		response := map[string]interface{}{
			"config":   cfg,
			"keywords": keywords,
			"topics":   string(topicsData),
			"anchors":  string(anchorsData),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method == http.MethodPost {
		var payload struct {
			Config   config.AppConfig `json:"config"`
			Keywords string           `json:"keywords"`
			Topics   string           `json:"topics"`
			Anchors  string           `json:"anchors"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Save YAML config
		if err := config.SaveConfig("config/collection.yml", &payload.Config); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Write text files
		if err := os.WriteFile(payload.Config.Keywords, []byte(payload.Keywords), 0644); err != nil {
			http.Error(w, "Failed to save keywords: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(payload.Config.Topics, []byte(payload.Topics), 0644); err != nil {
			http.Error(w, "Failed to save topics: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(payload.Config.Anchors, []byte(payload.Anchors), 0644); err != nil {
			http.Error(w, "Failed to save anchors: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write([]byte(`{"status": "success"}`))
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (s *APIServer) handleRunPipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.Lock()
	if s.status.Syncing {
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "pipeline already running"}`))
		return
	}
	s.status.Syncing = true
	s.status.Progress = 0
	s.status.Logs = []string{}
	s.mu.Unlock()

	s.addLog("[INFO] Pipeline synchronization initiated by web client.")

	go func() {
		cfg, err := config.LoadConfig("config/collection.yml")
		if err != nil {
			s.updateStatus(false, 0, "[ERROR] Failed to load config: "+err.Error())
			return
		}

		s.addLog("[INFO] Initializing openalex.DownloadPapers worker pool...")
		client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, cfg.Collection.PerPage, cfg.Collection.ConcurrentRequests, cfg.Collection.MaxRetries, cfg.Collection.RetryDelay)

		outputJSONL := filepath.Join(cfg.Output.JSONLDir, "collected_papers.jsonl")
		s.addLog("[INFO] Starting paper download to " + outputJSONL + "...")

		progressChan := make(chan int, 100)
		errChan := make(chan error, 1)

		go func() {
			errChan <- client.DownloadPapers(context.Background(), cfg, outputJSONL, progressChan)
		}()

		// Read progress events
		for {
			select {
			case p, ok := <-progressChan:
				if !ok {
					progressChan = nil
				} else {
					s.updateProgress(p)
				}
			case err := <-errChan:
				if err != nil {
					s.updateStatus(false, 0, "[ERROR] Download failed: "+err.Error())
					return
				}

				s.addLog("[SUCCESS] Ingestion completed. Ingested papers stored in JSONL.")
				s.addLog("[INFO] Starting DB conversion. Running dbMgr.CreateSchema...")

				dbPath := filepath.Join(cfg.Output.DBDir, "stratum.db")
				dbMgr, err := db.NewDBManager(dbPath)
				if err != nil {
					s.updateStatus(false, 0, "[ERROR] Failed to open DuckDB: "+err.Error())
					return
				}
				defer dbMgr.Close()

				if err := dbMgr.CreateSchema(); err != nil {
					s.updateStatus(false, 0, "[ERROR] Failed to create database schema: "+err.Error())
					return
				}

				s.addLog("[INFO] Importing collected_papers.jsonl into DuckDB dynamic schema...")
				stats, err := dbMgr.LoadJSONL(outputJSONL, nil)
				if err != nil {
					s.updateStatus(false, 0, "[ERROR] Failed to load JSONL: "+err.Error())
					return
				}

				s.addLog(fmt.Sprintf("[SUCCESS] Finished DuckDB load. Papers: %d, Authors: %d, Contributions: %d.", stats.Papers, stats.Authors, stats.Contributions))
				s.updateStatus(false, 100, "[SUCCESS] Sync cycle complete. Database is stable and query-ready.")
				return
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "started"}`))
}

func (s *APIServer) handlePipelineStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.mu.Lock()
	defer s.mu.Unlock()
	json.NewEncoder(w).Encode(s.status)
}
