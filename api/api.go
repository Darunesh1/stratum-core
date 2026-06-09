package api

import (
	"context"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
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
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/tfidf", s.handleTFIDF)
	mux.HandleFunc("/api/query/validate", s.handleQueryValidate)
	mux.HandleFunc("/api/openalex/count", s.handleOpenAlexCount)

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

	uploadDir := "data/uploads"
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

	filePath := filepath.Join("data/uploads", req.Filename)
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
			anchorPath := "config/anchor.txt"
			if cfg, err := config.LoadConfig("config/collection.yml"); err == nil && cfg.Anchors != "" {
				anchorPath = cfg.Anchors
			}
			os.WriteFile(anchorPath, []byte(anchorData), 0644)
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
	anchorPath := "config/anchor.txt"
	if cfg, err := config.LoadConfig("config/collection.yml"); err == nil && cfg.Anchors != "" {
		anchorPath = cfg.Anchors
	}
	if data, err := os.ReadFile(anchorPath); err == nil {
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
