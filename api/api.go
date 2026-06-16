package api

import (
	"context"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"stratum/config"
	"stratum/db"
	"stratum/openalex"
	"stratum/tfidf"

	"github.com/xuri/excelize/v2"
)

//go:embed dist/*
var frontendFS embed.FS

// ConfigRevision represents a versioned snapshot of keywords, topics, and anchors.
type ConfigRevision struct {
	Version   int    `json:"version"`
	Timestamp string `json:"timestamp"`
	Label     string `json:"label"`
	Keywords  string `json:"keywords"`
	Topics    string `json:"topics"`
	Anchors   string `json:"anchors"`
}

// PipelineStatus tracks the state and log output of the active ingestion pipeline.
type PipelineStatus struct {
	Syncing  bool     `json:"syncing"`
	Progress int      `json:"progress"`
	Logs     []string `json:"logs"`
}

// APIServer manages HTTP routes and coordinates tasks for the web client interface.
type APIServer struct {
	addr             string
	dbPath           string
	dbManagers       map[string]*db.DBManager
	pipelineStatuses map[string]*PipelineStatus
	projectMutexes   map[string]*sync.Mutex
	server           *http.Server
	mu               sync.Mutex
}

// NewAPIServer creates a new API server on the specified address.
func NewAPIServer(addr string, dbPath string) *APIServer {
	return &APIServer{
		addr:             addr,
		dbPath:           dbPath,
		dbManagers:       make(map[string]*db.DBManager),
		pipelineStatuses: make(map[string]*PipelineStatus),
		projectMutexes:   make(map[string]*sync.Mutex),
	}
}

// RegisterRoutes registers endpoints for dashboard metrics, query execution, files, and config.
func (s *APIServer) RegisterRoutes() error {
	mux := http.NewServeMux()

	// Register API endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/query", s.handleQuery)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/run-pipeline", s.handleRunPipeline)
	mux.HandleFunc("/api/pipeline/status", s.handlePipelineStatus)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/tfidf", s.handleTFIDF)
	mux.HandleFunc("/api/query/validate", s.handleQueryValidate)
	mux.HandleFunc("/api/openalex/count", s.handleOpenAlexCount)
	mux.HandleFunc("/api/projects", s.handleListProjects)
	mux.HandleFunc("/api/projects/create", s.handleCreateProject)

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
		Handler: loggingMiddleware(mux),
	}

	return nil
}

// Start starts the HTTP server asynchronously.
func (s *APIServer) Start() error {
	if s.server == nil {
		return fmt.Errorf("server not registered, call RegisterRoutes first")
	}
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[FATAL] HTTP server ListenAndServe failed: %v", err)
			os.Exit(1)
		}
	}()
	return nil
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)
		duration := time.Since(start)
		log.Printf("[HTTP] %s %s - %d (%s)", r.Method, r.URL.String(), lrw.statusCode, duration)
	})
}

// Stop gracefully shuts down the server.
func (s *APIServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, mgr := range s.dbManagers {
		mgr.Close()
	}
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func sanitizeProjectName(name string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	return reg.ReplaceAllString(name, "")
}

func (s *APIServer) getProjectPaths(project string) (ymlPath, keywordsPath, topicsPath, anchorsPath, dbPath, jsonlDir, dbDir, uploadsDir, historyPath string) {
	if project == "" || project == "default" {
		ymlPath = "config/collection.yml"
		keywordsPath = "config/keywords.txt"
		topicsPath = "config/topics.txt"
		anchorsPath = "config/anchor.txt"
		dbPath = s.dbPath
		jsonlDir = "data/jsonl"
		dbDir = "data"
		uploadsDir = "data/uploads"
		historyPath = "config/history.json"
		return
	}

	project = sanitizeProjectName(project)
	projDir := filepath.Join("projects", project)
	ymlPath = filepath.Join(projDir, "config", "collection.yml")
	keywordsPath = filepath.Join(projDir, "config", "keywords.txt")
	topicsPath = filepath.Join(projDir, "config", "topics.txt")
	anchorsPath = filepath.Join(projDir, "config", "anchor.txt")
	dbPath = filepath.Join(projDir, "data", "stratum.db")
	jsonlDir = filepath.Join(projDir, "data", "jsonl")
	dbDir = filepath.Join(projDir, "data")
	uploadsDir = filepath.Join(projDir, "data", "uploads")
	historyPath = filepath.Join(projDir, "config", "history.json")
	return
}

func (s *APIServer) ensureProjectDirs(project string) error {
	if project == "" || project == "default" {
		_ = os.MkdirAll("config", 0755)
		_ = os.MkdirAll("data/jsonl", 0755)
		_ = os.MkdirAll("data/uploads", 0755)
		return nil
	}

	project = sanitizeProjectName(project)
	projDir := filepath.Join("projects", project)
	if err := os.MkdirAll(filepath.Join(projDir, "config"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projDir, "data", "jsonl"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(projDir, "data", "uploads"), 0755); err != nil {
		return err
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func (s *APIServer) initializeProjectFiles(project string) error {
	if project == "" || project == "default" {
		return nil
	}
	project = sanitizeProjectName(project)
	ymlPath, keywordsPath, topicsPath, anchorsPath, _, _, _, _, _ := s.getProjectPaths(project)

	if _, err := os.Stat(ymlPath); os.IsNotExist(err) {
		_ = copyFile("config/collection.yml", ymlPath)
	}
	if _, err := os.Stat(keywordsPath); os.IsNotExist(err) {
		_ = copyFile("config/keywords.txt", keywordsPath)
	}
	if _, err := os.Stat(topicsPath); os.IsNotExist(err) {
		_ = copyFile("config/topics.txt", topicsPath)
	}
	if _, err := os.Stat(anchorsPath); os.IsNotExist(err) {
		_ = copyFile("config/anchor.txt", anchorsPath)
	}
	return nil
}

func (s *APIServer) getDBMgr(project string) (*db.DBManager, error) {
	if project == "" {
		project = "default"
	}
	project = sanitizeProjectName(project)

	s.mu.Lock()
	if mgr, ok := s.dbManagers[project]; ok {
		s.mu.Unlock()
		return mgr, nil
	}

	if s.projectMutexes == nil {
		s.projectMutexes = make(map[string]*sync.Mutex)
	}
	projMu, ok := s.projectMutexes[project]
	if !ok {
		projMu = &sync.Mutex{}
		s.projectMutexes[project] = projMu
	}
	s.mu.Unlock()

	projMu.Lock()
	defer projMu.Unlock()

	// Double-check after acquiring the project mutex
	s.mu.Lock()
	if mgr, ok := s.dbManagers[project]; ok {
		s.mu.Unlock()
		return mgr, nil
	}
	s.mu.Unlock()

	_, _, _, _, dbPath, _, _, _, _ := s.getProjectPaths(project)

	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	mgr, err := db.NewDBManager(dbPath)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.dbManagers[project] = mgr
	s.mu.Unlock()

	return mgr, nil
}

func (s *APIServer) getPipelineStatus(project string) *PipelineStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	if project == "" {
		project = "default"
	}
	project = sanitizeProjectName(project)

	status, ok := s.pipelineStatuses[project]
	if !ok {
		status = &PipelineStatus{
			Syncing:  false,
			Progress: 0,
			Logs:     []string{},
		}
		s.pipelineStatuses[project] = status
	}
	return status
}

func (s *APIServer) addLog(project string, msg string) {
	status := s.getPipelineStatus(project)
	s.mu.Lock()
	defer s.mu.Unlock()
	timestamp := time.Now().Format("15:04:05")
	status.Logs = append(status.Logs, fmt.Sprintf("[%s] %s", timestamp, msg))
}

func (s *APIServer) updateProgress(project string, p int) {
	status := s.getPipelineStatus(project)
	s.mu.Lock()
	defer s.mu.Unlock()
	status.Progress = p
	if status.Progress > 95 {
		status.Progress = 95
	}
	timestamp := time.Now().Format("15:04:05")
	status.Logs = append(status.Logs, fmt.Sprintf("[%s] [INFO] Download progress: %d papers fetched.", timestamp, p))
}

func (s *APIServer) updateStatus(project string, syncing bool, progress int, msg string) {
	status := s.getPipelineStatus(project)
	s.mu.Lock()
	defer s.mu.Unlock()
	status.Syncing = syncing
	status.Progress = progress
	timestamp := time.Now().Format("15:04:05")
	status.Logs = append(status.Logs, fmt.Sprintf("[%s] %s", timestamp, msg))
}

func loadConfigHistory(path string) ([]ConfigRevision, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ConfigRevision{}, nil
		}
		return nil, err
	}
	var list []ConfigRevision
	if err := json.Unmarshal(data, &list); err != nil {
		return []ConfigRevision{}, nil
	}
	return list, nil
}

func saveConfigHistory(path string, list []ConfigRevision) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *APIServer) appendConfigRevision(historyPath, keywords, topics, anchors, label string) error {
	list, err := loadConfigHistory(historyPath)
	if err != nil {
		list = []ConfigRevision{}
	}

	nextVersion := 1
	if len(list) > 0 {
		nextVersion = list[len(list)-1].Version + 1
	}

	if label == "" {
		label = fmt.Sprintf("Revision #%d", nextVersion)
	}

	rev := ConfigRevision{
		Version:   nextVersion,
		Timestamp: time.Now().Format(time.RFC3339),
		Label:     label,
		Keywords:  keywords,
		Topics:    topics,
		Anchors:   anchors,
	}

	list = append(list, rev)
	return saveConfigHistory(historyPath, list)
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "online"}`))
}

func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	project := r.URL.Query().Get("project")
	dbMgr, err := s.getDBMgr(project)
	if err != nil {
		http.Error(w, "Database manager error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	stats, err := dbMgr.GetDashboardStats()
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

	project := r.URL.Query().Get("project")
	dbMgr, err := s.getDBMgr(project)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	results, err := dbMgr.RunQuery(body.Query)
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
		project := r.URL.Query().Get("project")
		ymlPath, keywordsPath, topicsPath, anchorsPath, _, jsonlDir, dbDir, _, historyPath := s.getProjectPaths(project)
		s.ensureProjectDirs(project)
		s.initializeProjectFiles(project)

		cfg, err := config.LoadConfig(ymlPath)
		if err != nil {
			// fallback
			cfg, err = config.LoadConfig("config/collection.yml")
			if err != nil {
				http.Error(w, "Failed to load config: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		cfg.Keywords = keywordsPath
		cfg.Topics = topicsPath
		cfg.Anchors = anchorsPath
		cfg.Output.JSONLDir = jsonlDir
		cfg.Output.DBDir = dbDir

		keywords, _ := config.GetKeywords(cfg.Keywords)
		topicsData, _ := os.ReadFile(cfg.Topics)
		anchorsData, _ := os.ReadFile(cfg.Anchors)
		historyList, _ := loadConfigHistory(historyPath)
		if historyList == nil {
			historyList = []ConfigRevision{}
		}

		response := map[string]interface{}{
			"config":   cfg,
			"keywords": keywords,
			"topics":   string(topicsData),
			"anchors":  string(anchorsData),
			"history":  historyList,
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
			Label    string           `json:"label"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		project := r.URL.Query().Get("project")
		ymlPath, keywordsPath, topicsPath, anchorsPath, _, jsonlDir, dbDir, _, historyPath := s.getProjectPaths(project)
		s.ensureProjectDirs(project)

		payload.Config.Keywords = keywordsPath
		payload.Config.Topics = topicsPath
		payload.Config.Anchors = anchorsPath
		payload.Config.Output.JSONLDir = jsonlDir
		payload.Config.Output.DBDir = dbDir

		// Save YAML config
		if err := config.SaveConfig(ymlPath, &payload.Config); err != nil {
			http.Error(w, "Failed to save config: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Write text files
		if err := os.WriteFile(keywordsPath, []byte(payload.Keywords), 0644); err != nil {
			http.Error(w, "Failed to save keywords: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(topicsPath, []byte(payload.Topics), 0644); err != nil {
			http.Error(w, "Failed to save topics: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := os.WriteFile(anchorsPath, []byte(payload.Anchors), 0644); err != nil {
			http.Error(w, "Failed to save anchors: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Append revision to history
		_ = s.appendConfigRevision(historyPath, payload.Keywords, payload.Topics, payload.Anchors, payload.Label)

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

	project := r.URL.Query().Get("project")
	status := s.getPipelineStatus(project)

	s.mu.Lock()
	if status.Syncing {
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "pipeline already running"}`))
		return
	}
	status.Syncing = true
	status.Progress = 0
	status.Logs = []string{}
	s.mu.Unlock()

	s.addLog(project, "[INFO] Pipeline synchronization initiated by web client.")

	go func() {
		ymlPath, _, _, _, dbPath, jsonlDir, dbDir, _, _ := s.getProjectPaths(project)
		cfg, err := config.LoadConfig(ymlPath)
		if err != nil {
			s.updateStatus(project, false, 0, "[ERROR] Failed to load config: "+err.Error())
			return
		}

		cfg.Keywords = filepath.Join(filepath.Dir(ymlPath), "keywords.txt")
		cfg.Topics = filepath.Join(filepath.Dir(ymlPath), "topics.txt")
		cfg.Anchors = filepath.Join(filepath.Dir(ymlPath), "anchor.txt")
		cfg.Output.JSONLDir = jsonlDir
		cfg.Output.DBDir = dbDir

		s.addLog(project, "[INFO] Initializing openalex.DownloadPapers worker pool...")
		client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, cfg.Collection.PerPage, cfg.Collection.ConcurrentRequests, cfg.Collection.MaxRetries, cfg.Collection.RetryDelay)

		outputJSONL := filepath.Join(cfg.Output.JSONLDir, "collected_papers.jsonl")
		s.addLog(project, "[INFO] Starting paper download to " + outputJSONL + "...")

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
					s.updateProgress(project, p)
				}
			case err := <-errChan:
				if err != nil {
					s.updateStatus(project, false, 0, "[ERROR] Download failed: "+err.Error())
					return
				}

				s.addLog(project, "[SUCCESS] Ingestion completed. Ingested papers stored in JSONL.")
				s.addLog(project, "[INFO] Starting DB conversion. Running dbMgr.CreateSchema...")

				dbMgr, err := db.NewDBManager(dbPath)
				if err != nil {
					s.updateStatus(project, false, 0, "[ERROR] Failed to open DuckDB: "+err.Error())
					return
				}
				defer dbMgr.Close()

				if err := dbMgr.CreateSchema(); err != nil {
					s.updateStatus(project, false, 0, "[ERROR] Failed to create database schema: "+err.Error())
					return
				}

				s.addLog(project, "[INFO] Importing collected_papers.jsonl into DuckDB dynamic schema...")
				stats, err := dbMgr.LoadJSONL(outputJSONL, nil)
				if err != nil {
					s.updateStatus(project, false, 0, "[ERROR] Failed to load JSONL: "+err.Error())
					return
				}

				s.addLog(project, fmt.Sprintf("[SUCCESS] Finished DuckDB load. Papers: %d, Authors: %d, Contributions: %d.", stats.Papers, stats.Authors, stats.Contributions))
				s.updateStatus(project, false, 100, "[SUCCESS] Sync cycle complete. Database is stable and query-ready.")
				return
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "started"}`))
}

func (s *APIServer) handlePipelineStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	project := r.URL.Query().Get("project")
	status := s.getPipelineStatus(project)
	s.mu.Lock()
	defer s.mu.Unlock()
	json.NewEncoder(w).Encode(status)
}

func (s *APIServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// Limit upload size to 10MB
	r.ParseMultipartForm(10 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve file from form: " + err.Error()})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Unsupported file format. Please upload a .csv, .xlsx, or .xls file."})
		return
	}

	project := r.URL.Query().Get("project")
	_, _, _, _, _, _, _, uploadDir, _ := s.getProjectPaths(project)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create uploads directory: " + err.Error()})
		return
	}

	// Save file to disk
	safeName := fmt.Sprintf("upload_%d%s", time.Now().UnixNano(), ext)
	filePath := filepath.Join(uploadDir, safeName)
	dst, err := os.Create(filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create destination file: " + err.Error()})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to write file to disk: " + err.Error()})
		return
	}

	// Extract headers
	var headers []string
	if ext == ".csv" {
		headers, err = parseCSVHeaders(filePath)
	} else {
		headers, err = parseExcelHeaders(filePath)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to parse file headers: " + err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"filename": safeName,
		"columns":  headers,
	})
}

func parseCSVHeaders(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	return headers, nil
}

func parseExcelHeaders(filePath string) ([]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("empty sheet")
	}
	return rows[0], nil
}

func (s *APIServer) handleTFIDF(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Filename       string  `json:"filename"`
		TitleColumn    string  `json:"title_column"`
		AbstractColumn string  `json:"abstract_column"`
		DOIColumn      string  `json:"doi_column"`
		TopN           int     `json:"top_n"`
		NgramMin       int     `json:"ngram_min"`
		NgramMax       int     `json:"ngram_max"`
		MinDF          int     `json:"min_df"`
		MaxDF          float64 `json:"max_df"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.Filename == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "filename parameter is required"})
		return
	}

	// Apply defaults if empty
	if req.TopN <= 0 {
		req.TopN = 50
	}
	if req.NgramMin <= 0 {
		req.NgramMin = 2
	}
	if req.NgramMax <= 0 {
		req.NgramMax = 3
	}
	if req.MinDF <= 0 {
		req.MinDF = 2
	}
	if req.MaxDF <= 0.0 {
		req.MaxDF = 0.85
	}

	project := r.URL.Query().Get("project")
	_, _, _, anchorsPath, _, _, _, uploadsDir, _ := s.getProjectPaths(project)

	filePath := filepath.Join(uploadsDir, req.Filename)
	ext := strings.ToLower(filepath.Ext(filePath))

	var docs []string
	var err error
	if ext == ".csv" {
		docs, err = loadCSVDocuments(filePath, req.TitleColumn, req.AbstractColumn)
	} else {
		docs, err = loadExcelDocuments(filePath, req.TitleColumn, req.AbstractColumn)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to extract documents: " + err.Error()})
		return
	}

	// Extract DOIs if DOI column is provided
	var dois []string
	if req.DOIColumn != "" {
		if ext == ".csv" {
			dois, err = extractDOIsFromCSV(filePath, req.DOIColumn)
		} else {
			dois, err = extractDOIsFromExcel(filePath, req.DOIColumn)
		}
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to extract DOIs: " + err.Error()})
			return
		}

		if len(dois) > 0 {
			anchorData := strings.Join(dois, "\n") + "\n"
			_ = os.WriteFile(anchorsPath, []byte(anchorData), 0644)
		}
	}

	keywords := tfidf.ExtractKeywords(docs, req.NgramMin, req.NgramMax, req.MinDF, req.MaxDF, req.TopN)
	if keywords == nil {
		keywords = []tfidf.ScoredTerm{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"keywords":      keywords,
		"anchors_count": len(dois),
	})
}

func loadCSVDocuments(filePath, titleCol, abstractCol string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("empty file or no data rows")
	}

	headers := rows[0]
	titleIdx, abstractIdx := -1, -1
	for idx, h := range headers {
		if strings.EqualFold(h, titleCol) {
			titleIdx = idx
		}
		if strings.EqualFold(h, abstractCol) {
			abstractIdx = idx
		}
	}

	if titleIdx == -1 && abstractIdx == -1 {
		return nil, fmt.Errorf("neither title column %q nor abstract column %q was found in headers", titleCol, abstractCol)
	}

	var docs []string
	for _, row := range rows[1:] {
		var title, abstract string
		if titleIdx != -1 && titleIdx < len(row) {
			title = strings.TrimSpace(row[titleIdx])
		}
		if abstractIdx != -1 && abstractIdx < len(row) {
			abstract = strings.TrimSpace(row[abstractIdx])
		}
		combined := strings.TrimSpace(title + ". " + abstract)
		if len(combined) >= 20 {
			docs = append(docs, combined)
		}
	}
	return docs, nil
}

func loadExcelDocuments(filePath, titleCol, abstractCol string) ([]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("empty sheet or no data rows")
	}

	headers := rows[0]
	titleIdx, abstractIdx := -1, -1
	for idx, h := range headers {
		if strings.EqualFold(h, titleCol) {
			titleIdx = idx
		}
		if strings.EqualFold(h, abstractCol) {
			abstractIdx = idx
		}
	}

	if titleIdx == -1 && abstractIdx == -1 {
		return nil, fmt.Errorf("neither title column %q nor abstract column %q was found in headers", titleCol, abstractCol)
	}

	var docs []string
	for _, row := range rows[1:] {
		var title, abstract string
		if titleIdx != -1 && titleIdx < len(row) {
			title = strings.TrimSpace(row[titleIdx])
		}
		if abstractIdx != -1 && abstractIdx < len(row) {
			abstract = strings.TrimSpace(row[abstractIdx])
		}
		combined := strings.TrimSpace(title + ". " + abstract)
		if len(combined) >= 20 {
			docs = append(docs, combined)
		}
	}
	return docs, nil
}

func (s *APIServer) handleQueryValidate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query string `json:"query"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	errors := openalex.ValidateKeywords(req.Query)
	if len(errors) > 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":  false,
			"errors": errors,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":  true,
		"errors": []string{},
	})
}

func (s *APIServer) handleOpenAlexCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Query    string   `json:"query"`
		Keys     []string `json:"keys"`
		Email    string   `json:"email"`
		DateFrom string   `json:"date_from"`
		DateTo   string   `json:"date_to"`
		DocTypes []string `json:"doc_types"`
		Topics   []string `json:"topics"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.Email == "" {
		req.Email = "your@email.com"
	}
	if req.DateFrom == "" {
		req.DateFrom = "2003-01-01"
	}
	if req.DateTo == "" {
		req.DateTo = "2024-12-31"
	}

	// Instantiate client
	client := openalex.NewClient(req.Keys, req.Email, 200, 1, 3, 1)

	// Build the API filter query
	parts := []string{"title_and_abstract.search:" + req.Query}
	if len(req.Topics) > 0 {
		parts = append(parts, "primary_topic.id:"+strings.Join(req.Topics, "|"))
	}
	parts = append(parts, "from_publication_date:"+req.DateFrom)
	parts = append(parts, "to_publication_date:"+req.DateTo)
	if len(req.DocTypes) > 0 {
		parts = append(parts, "type:"+strings.Join(req.DocTypes, "|"))
	}
	filter := strings.Join(parts, ",")

	count, err := client.GetTotalCount(r.Context(), filter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "OpenAlex request failed: " + err.Error()})
		return
	}

	// Load anchors from config
	var anchors []string
	project := r.URL.Query().Get("project")
	_, _, _, anchorsPath, _, _, _, _, _ := s.getProjectPaths(project)

	if data, err := os.ReadFile(anchorsPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				norm := normalizeDOI(line)
				if norm != "" {
					anchors = append(anchors, norm)
				}
			}
		}
	}

	// Run anchor check coverage
	var matchedCount int
	var missingDOIs []string
	if len(anchors) > 0 {
		batchSize := 10
		matchedSet := make(map[string]bool)

		for i := 0; i < len(anchors); i += batchSize {
			end := i + batchSize
			if end > len(anchors) {
				end = len(anchors)
			}
			batch := anchors[i:end]
			batchFilter := strings.Join(batch, "|")

			// Combine filter: queryFilter + ",doi:" + batchFilter
			combinedFilter := filter + ",doi:" + batchFilter

			// Query OpenAlex works for matching DOIs in this batch
			resp, err := client.FetchPage(r.Context(), combinedFilter, "*")
			if err == nil && resp != nil {
				for _, w := range resp.Results {
					norm := normalizeDOI(w.DOI)
					if norm != "" {
						matchedSet[norm] = true
					}
				}
			}
		}

		for _, doi := range anchors {
			if matchedSet[doi] {
				matchedCount++
			} else {
				missingDOIs = append(missingDOIs, doi)
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":           count,
		"anchors_total":   len(anchors),
		"anchors_matched": matchedCount,
		"anchors_missing": missingDOIs,
	})
}

var doiPrefixRe = regexp.MustCompile(`(?i)^(?:https?://(?:dx\.)?doi\.org/|doi:)`)
var doiRe = regexp.MustCompile(`(?i)^10\.\d{4,9}/\S+$`)

func normalizeDOI(val string) string {
	candidate := strings.TrimSpace(val)
	if candidate == "" {
		return ""
	}
	candidate = doiPrefixRe.ReplaceAllString(candidate, "")
	candidate = strings.TrimSpace(candidate)
	candidate = strings.ToLower(candidate)
	if doiRe.MatchString(candidate) {
		return candidate
	}
	return ""
}

func extractDOIsFromCSV(filePath, doiCol string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, nil
	}

	headers := rows[0]
	doiIdx := -1
	for idx, h := range headers {
		if strings.EqualFold(h, doiCol) {
			doiIdx = idx
			break
		}
	}
	if doiIdx == -1 {
		return nil, fmt.Errorf("doi column %q not found in headers", doiCol)
	}

	var dois []string
	seen := make(map[string]bool)
	for _, row := range rows[1:] {
		if doiIdx < len(row) {
			rawDOI := strings.TrimSpace(row[doiIdx])
			norm := normalizeDOI(rawDOI)
			if norm != "" && !seen[norm] {
				seen[norm] = true
				dois = append(dois, norm)
			}
		}
	}
	return dois, nil
}

func extractDOIsFromExcel(filePath, doiCol string) ([]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, nil
	}

	headers := rows[0]
	doiIdx := -1
	for idx, h := range headers {
		if strings.EqualFold(h, doiCol) {
			doiIdx = idx
			break
		}
	}
	if doiIdx == -1 {
		return nil, fmt.Errorf("doi column %q not found in headers", doiCol)
	}

	var dois []string
	seen := make(map[string]bool)
	for _, row := range rows[1:] {
		if doiIdx < len(row) {
			rawDOI := strings.TrimSpace(row[doiIdx])
			norm := normalizeDOI(rawDOI)
			if norm != "" && !seen[norm] {
				seen[norm] = true
				dois = append(dois, norm)
			}
		}
	}
	return dois, nil
}

func (s *APIServer) handleListProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	projects := []string{"default"}

	if entries, err := os.ReadDir("projects"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				name := entry.Name()
				if name == sanitizeProjectName(name) {
					projects = append(projects, name)
				}
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"projects": projects,
	})
}

func (s *APIServer) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body: " + err.Error()})
		return
	}

	name := sanitizeProjectName(req.Name)
	if name == "" || name == "default" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid project name"})
		return
	}

	if err := s.ensureProjectDirs(name); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create project directories: " + err.Error()})
		return
	}

	if err := s.initializeProjectFiles(name); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to initialize project files: " + err.Error()})
		return
	}

	// Create initial history version 1
	ymlPath, _, _, _, _, _, _, _, historyPath := s.getProjectPaths(name)
	cfg, err := config.LoadConfig(ymlPath)
	var keywords, topics, anchors string
	if err == nil {
		keywordsData, _ := config.GetKeywords(cfg.Keywords)
		keywords = keywordsData
		td, _ := os.ReadFile(cfg.Topics)
		topics = string(td)
		ad, _ := os.ReadFile(cfg.Anchors)
		anchors = string(ad)
	}
	_ = s.appendConfigRevision(historyPath, keywords, topics, anchors, "Project Created")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"name":   name,
	})
}
