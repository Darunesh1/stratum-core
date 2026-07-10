package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"stratum/config"
	"stratum/docs"
	"stratum/impute"
	"stratum/openalex"
	"stratum/workflow"
)

// MCP Types mapped from standalone server
type ValidateArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"Optional path to the config file or config.db"`
}

type ValidateResult struct {
	Valid  bool     `json:"valid" jsonschema:"Indicates if keywords and topics are structurally valid"`
	Errors []string `json:"errors" jsonschema:"List of validation error messages"`
}

type SearchArgs struct {
	ConfigPath   string `json:"config_path,omitempty" jsonschema:"Optional path to the config file or config.db"`
	CheckAnchors bool   `json:"check_anchors,omitempty" jsonschema:"Optional flag to check anchor DOI coverage"`
}

type SearchResult struct {
	TotalCount     int      `json:"total_count" jsonschema:"Total matching papers"`
	AnchorsTotal   int      `json:"anchors_total,omitempty"`
	AnchorsMatched int      `json:"anchors_matched,omitempty"`
	AnchorsMissing []string `json:"anchors_missing,omitempty"`
}

type DownloadArgs struct {
	ConfigPath  string `json:"config_path,omitempty" jsonschema:"Optional path to the config file or config.db"`
	OutputJSONL string `json:"output_jsonl,omitempty" jsonschema:"Optional path to write downloaded JSONL"`
}

type DownloadResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ConvertDBArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"Optional path to the config file or config.db"`
	JSONLPath  string `json:"jsonl_path,omitempty" jsonschema:"Optional path to input JSONL"`
}

type ConvertDBResult struct {
	Status        string `json:"status"`
	PapersLoaded  int    `json:"papers_loaded"`
	AuthorsLoaded int    `json:"authors_loaded"`
	InstsLoaded   int    `json:"institutions_loaded"`
	Errors        int    `json:"errors"`
}

type ImputeArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"Optional path to the config file or config.db"`
	Pipeline   string `json:"pipeline,omitempty" jsonschema:"Pipeline stage to execute: crossref, llm, pdf, or all"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Limit for PDF extraction"`
}

type ImputeResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type GetTopicsArgs struct {
	ConfigPath string `json:"config_path,omitempty" jsonschema:"Optional path to the config file or config.db"`
	Details    bool   `json:"details,omitempty"`
}

type GetTopicsResult struct {
	Markdown string `json:"markdown"`
}

func (s *APIServer) resolveProjectFromConfigPath(configPath string) string {
	if configPath == "" {
		return "default"
	}
	cleaned := filepath.ToSlash(filepath.Clean(configPath))
	parts := strings.Split(cleaned, "/")
	for i := 0; i < len(parts)-2; i++ {
		if parts[i] == "projects" {
			return sanitizeProjectName(parts[i+1])
		}
	}
	return "default"
}

func (s *APIServer) RegisterMCPTools() error {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "validate",
		Description: "Validate the keywords syntax and check if configured topics exist in OpenAlex.",
	}, s.handleValidate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search",
		Description: "Query OpenAlex to return the total count of academic papers matching current configuration filters.",
	}, s.handleSearch)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "download",
		Description: "Download papers matching configuration filters concurrently and save them to a JSONL file.",
	}, s.handleDownload)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "convert_db",
		Description: "Import downloaded JSONL paper records into DuckDB with schema initialization.",
	}, s.handleConvertDB)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "impute",
		Description: "Impute missing institution and country metadata using Crossref, LLMs, and PDF text extraction.",
	}, s.handleImpute)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_topics",
		Description: "Fetch the distribution of research topics and paper counts matching the keyword filters from OpenAlex.",
	}, s.handleGetTopics)

	return nil
}

func (s *APIServer) handleValidate(ctx context.Context, req *mcp.CallToolRequest, args ValidateArgs) (*mcp.CallToolResult, ValidateResult, error) {
	project := s.resolveProjectFromConfigPath(args.ConfigPath)
	configDBPath, _, _, _, _ := s.getProjectPaths(project)

	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		errStr := "failed to load config: " + err.Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, ValidateResult{Valid: false, Errors: []string{errStr}}, nil
	}

	var errors []string
	keywords := cfg.Keywords
	if keywords != "" {
		kwErrs := openalex.ValidateKeywords(keywords)
		errors = append(errors, kwErrs...)
	}

	topics := cfg.Topics
	if len(topics) > 0 {
		var validTopics []string
		for _, topic := range topics {
			if !openalex.ValidateTopicFormat(topic) {
				errors = append(errors, "invalid topic format: "+topic)
			} else {
				validTopics = append(validTopics, topic)
			}
		}

		if len(validTopics) > 0 {
			client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, cfg.Collection.PerPage, cfg.Collection.ConcurrentRequests, cfg.Collection.MaxRetries, cfg.Collection.RetryDelay)
			existsMap, err := openalex.ValidateTopicsExist(ctx, client, validTopics)
			if err != nil {
				errors = append(errors, "failed to check topics existence: "+err.Error())
			} else {
				for _, topic := range validTopics {
					if !existsMap[topic] {
						errors = append(errors, "topic does not exist in OpenAlex: "+topic)
					}
				}
			}
		}
	}

	valid := len(errors) == 0
	msg := fmt.Sprintf("Validation complete. Valid: %t. Errors: %d.", valid, len(errors))
	if !valid {
		msg += " Errors: " + strings.Join(errors, "; ")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, ValidateResult{Valid: valid, Errors: errors}, nil
}

func (s *APIServer) handleSearch(ctx context.Context, req *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, SearchResult, error) {
	project := s.resolveProjectFromConfigPath(args.ConfigPath)
	configDBPath, _, _, _, _ := s.getProjectPaths(project)

	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		errStr := "failed to load config: " + err.Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, SearchResult{}, nil
	}

	keywords := cfg.Keywords
	if errs := openalex.ValidateKeywords(keywords); len(errs) > 0 {
		errStr := "keyword validation failed: " + strings.Join(errs, "; ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, SearchResult{}, nil
	}
	topics := cfg.Topics

	client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, cfg.Collection.PerPage, cfg.Collection.ConcurrentRequests, cfg.Collection.MaxRetries, cfg.Collection.RetryDelay)

	parts := []string{"title_and_abstract.search:" + keywords}
	if len(topics) > 0 {
		parts = append(parts, "primary_topic.id:"+strings.Join(topics, "|"))
	}
	dateFrom := cfg.Filters.DateFrom
	if dateFrom == "" {
		dateFrom = "2003-01-01"
	}
	dateTo := cfg.Filters.DateTo
	if dateTo == "" {
		dateTo = "2024-12-31"
	}
	parts = append(parts, "from_publication_date:"+dateFrom)
	parts = append(parts, "to_publication_date:"+dateTo)
	if len(cfg.Filters.DocTypes) > 0 {
		parts = append(parts, "type:"+strings.Join(cfg.Filters.DocTypes, "|"))
	}
	filter := strings.Join(parts, ",")

	count, err := client.GetTotalCount(ctx, filter)
	if err != nil {
		errStr := "OpenAlex search request failed: " + err.Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, SearchResult{}, nil
	}

	var anchors []string
	var matchedCount int
	var missingDOIs []string

	if args.CheckAnchors {
		for _, a := range cfg.Anchors {
			norm := normalizeDOI(a)
			if norm != "" {
				anchors = append(anchors, norm)
			}
		}

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
				combinedFilter := filter + ",doi:" + batchFilter

				resp, err := client.FetchPage(ctx, combinedFilter, "*")
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
	}

	text := fmt.Sprintf("Found %d papers matching configuration filters in OpenAlex.", count)
	if args.CheckAnchors {
		text += fmt.Sprintf(" Anchor match coverage: %d/%d matches.", matchedCount, len(anchors))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, SearchResult{
		TotalCount:     count,
		AnchorsTotal:   len(anchors),
		AnchorsMatched: matchedCount,
		AnchorsMissing: missingDOIs,
	}, nil
}

func (s *APIServer) handleDownload(ctx context.Context, req *mcp.CallToolRequest, args DownloadArgs) (*mcp.CallToolResult, DownloadResult, error) {
	project := s.resolveProjectFromConfigPath(args.ConfigPath)
	configDBPath, _, jsonlDir, _, _ := s.getProjectPaths(project)

	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		errStr := "failed to load config: " + err.Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, DownloadResult{Status: "error", Message: errStr}, nil
	}

	outputJSONL := args.OutputJSONL
	if outputJSONL == "" {
		if err := os.MkdirAll(jsonlDir, 0755); err != nil {
			errStr := "failed to create JSONL output directory: " + err.Error()
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
				IsError: true,
			}, DownloadResult{Status: "error", Message: errStr}, nil
		}
		outputJSONL = filepath.Join(jsonlDir, "collected_papers.jsonl")
	} else {
		if err := os.MkdirAll(filepath.Dir(outputJSONL), 0755); err != nil {
			errStr := "failed to create output directory: " + err.Error()
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
				IsError: true,
			}, DownloadResult{Status: "error", Message: errStr}, nil
		}
	}

	status := s.getPipelineStatus(project)

	s.mu.Lock()
	if status.Syncing {
		s.mu.Unlock()
		errStr := "pipeline already running"
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, DownloadResult{Status: "error", Message: errStr}, nil
	}
	status.Syncing = true
	status.Progress = 0
	status.Logs = []string{}
	s.mu.Unlock()

	s.addLog(project, "[MCP] Ingestion pipeline started by AI agent.")

	client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, cfg.Collection.PerPage, cfg.Collection.ConcurrentRequests, cfg.Collection.MaxRetries, cfg.Collection.RetryDelay)

	progressChan := make(chan int, 100)
	go func() {
		for p := range progressChan {
			s.updateProgress(project, p)
		}
	}()

	err = client.DownloadPapers(ctx, cfg, outputJSONL, progressChan)
	close(progressChan)

	s.mu.Lock()
	status.Syncing = false
	s.mu.Unlock()

	if err != nil {
		errStr := "download papers failed: " + err.Error()
		s.addLog(project, "[ERROR] [MCP] Download failed: "+err.Error())
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, DownloadResult{Status: "error", Message: errStr}, nil
	}

	msg := fmt.Sprintf("Download complete. Papers saved to %s.", outputJSONL)
	s.addLog(project, "[SUCCESS] [MCP] Ingestion completed. Papers stored in JSONL.")
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, DownloadResult{Status: "success", Message: msg}, nil
}

func (s *APIServer) handleConvertDB(ctx context.Context, req *mcp.CallToolRequest, args ConvertDBArgs) (*mcp.CallToolResult, ConvertDBResult, error) {
	project := s.resolveProjectFromConfigPath(args.ConfigPath)
	_, papersDBPath, jsonlDir, _, _ := s.getProjectPaths(project)

	jsonlPath := args.JSONLPath
	if jsonlPath == "" {
		jsonlPath = filepath.Join(jsonlDir, "collected_papers.jsonl")
	}

	status := s.getPipelineStatus(project)

	s.mu.Lock()
	if status.Syncing {
		s.mu.Unlock()
		errStr := "pipeline already running"
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, ConvertDBResult{Status: "error"}, nil
	}
	status.Syncing = true
	status.Progress = 0
	s.mu.Unlock()

	s.addLog(project, "[MCP] DB conversion initiated by AI agent.")

	dbMgr, err := s.getDBMgr(project)
	if err != nil {
		s.mu.Lock()
		status.Syncing = false
		s.mu.Unlock()
		errStr := "failed to get DB manager: " + err.Error()
		s.addLog(project, "[ERROR] [MCP] Failed to open DuckDB: "+err.Error())
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, ConvertDBResult{Status: "error"}, nil
	}

	s.addLog(project, "[MCP] Initializing schema...")
	if err := dbMgr.CreateSchema(); err != nil {
		s.mu.Lock()
		status.Syncing = false
		s.mu.Unlock()
		errStr := "failed to create schema: " + err.Error()
		s.addLog(project, "[ERROR] [MCP] Failed to create database schema: "+err.Error())
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, ConvertDBResult{Status: "error"}, nil
	}

	s.addLog(project, "[MCP] Loading JSONL data...")
	progressChan := make(chan int, 100)
	go func() {
		for p := range progressChan {
			s.updateProgress(project, p)
		}
	}()

	stats, err := dbMgr.LoadJSONL(jsonlPath, progressChan)
	close(progressChan)

	s.mu.Lock()
	status.Syncing = false
	s.mu.Unlock()

	if err != nil {
		errStr := "failed to import JSONL into DuckDB: " + err.Error()
		s.addLog(project, "[ERROR] [MCP] Failed to load JSONL: "+err.Error())
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, ConvertDBResult{Status: "error"}, nil
	}

	msg := fmt.Sprintf("Import complete. Loaded %d papers, %d authors, %d institutions into %s.", stats.Papers, stats.Authors, stats.Institutions, papersDBPath)
	s.addLog(project, fmt.Sprintf("[SUCCESS] [MCP] Finished DuckDB load. Papers: %d, Authors: %d, Contributions: %d.", stats.Papers, stats.Authors, stats.Contributions))
	s.addLog(project, "[SUCCESS] [MCP] Sync cycle complete. Database is stable and query-ready.")

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, ConvertDBResult{
		Status:        "success",
		PapersLoaded:  stats.Papers,
		AuthorsLoaded: stats.Authors,
		InstsLoaded:   stats.Institutions,
		Errors:        stats.Errors,
	}, nil
}

func (s *APIServer) handleImpute(ctx context.Context, req *mcp.CallToolRequest, args ImputeArgs) (*mcp.CallToolResult, ImputeResult, error) {
	project := s.resolveProjectFromConfigPath(args.ConfigPath)
	configDBPath, papersDBPath, _, _, _ := s.getProjectPaths(project)

	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		errStr := "failed to load config: " + err.Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, ImputeResult{Status: "error", Message: errStr}, nil
	}

	status := s.getPipelineStatus(project)

	s.mu.Lock()
	if status.Syncing {
		s.mu.Unlock()
		errStr := "pipeline already running"
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, ImputeResult{Status: "error", Message: errStr}, nil
	}
	status.Syncing = true
	status.Progress = 0
	s.mu.Unlock()

	s.addLog(project, "[MCP] Imputation pipeline initiated by AI agent.")

	engine := impute.NewImputationEngine(papersDBPath)

	pipeline := strings.ToLower(args.Pipeline)
	if pipeline == "" {
		pipeline = "all"
	}

	var results []string
	progressChan := make(chan int, 100)
	go func() {
		for range progressChan {
			// drain or update progress
		}
	}()

	if pipeline == "crossref" || pipeline == "all" {
		s.addLog(project, "[MCP] Running CrossRef metadata imputation...")
		if err := engine.ImputeCrossRef(ctx, progressChan); err != nil {
			results = append(results, "CrossRef failed: "+err.Error())
			s.addLog(project, "[ERROR] [MCP] CrossRef failed: "+err.Error())
		} else {
			results = append(results, "CrossRef imputation complete.")
			s.addLog(project, "[SUCCESS] [MCP] CrossRef imputation completed.")
		}
	}

	if pipeline == "llm" || pipeline == "all" {
		s.addLog(project, "[MCP] Running LLM affiliation metadata imputation...")
		provider := cfg.LLM.Provider
		model := cfg.LLM.Model
		baseURL := cfg.LLM.BaseURL
		if err := engine.ImputeLLM(ctx, provider, model, baseURL, progressChan); err != nil {
			results = append(results, "LLM imputation failed: "+err.Error())
			s.addLog(project, "[ERROR] [MCP] LLM imputation failed: "+err.Error())
		} else {
			results = append(results, "LLM imputation complete.")
			s.addLog(project, "[SUCCESS] [MCP] LLM imputation completed.")
		}
	}

	if pipeline == "pdf" || pipeline == "all" {
		s.addLog(project, "[MCP] Running PDF metadata extraction and imputation...")
		provider := cfg.LLM.Provider
		model := cfg.LLM.Model
		baseURL := cfg.LLM.BaseURL
		limit := args.Limit
		if limit <= 0 {
			limit = 10
		}
		if err := engine.ImputePDF(ctx, provider, model, baseURL, limit, progressChan); err != nil {
			results = append(results, "PDF imputation failed: "+err.Error())
			s.addLog(project, "[ERROR] [MCP] PDF imputation failed: "+err.Error())
		} else {
			results = append(results, "PDF imputation complete.")
			s.addLog(project, "[SUCCESS] [MCP] PDF imputation completed.")
		}
	}

	close(progressChan)

	s.mu.Lock()
	status.Syncing = false
	s.mu.Unlock()

	summary := strings.Join(results, "\n")
	s.addLog(project, "[SUCCESS] [MCP] Imputation pipeline finished.")

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: summary}},
	}, ImputeResult{Status: "success", Message: summary}, nil
}

func (s *APIServer) handleGetTopics(ctx context.Context, req *mcp.CallToolRequest, args GetTopicsArgs) (*mcp.CallToolResult, GetTopicsResult, error) {
	project := s.resolveProjectFromConfigPath(args.ConfigPath)
	configDBPath, _, _, _, _ := s.getProjectPaths(project)

	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		errStr := "failed to load config: " + err.Error()
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, GetTopicsResult{Markdown: ""}, nil
	}

	keywords := cfg.Keywords
	if errs := openalex.ValidateKeywords(keywords); len(errs) > 0 {
		errStr := "keyword validation failed: " + strings.Join(errs, "; ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
			IsError: true,
		}, GetTopicsResult{Markdown: ""}, nil
	}
	topics := cfg.Topics

	client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, cfg.Collection.PerPage, cfg.Collection.ConcurrentRequests, cfg.Collection.MaxRetries, cfg.Collection.RetryDelay)

	parts := []string{"title_and_abstract.search:" + keywords}
	if len(topics) > 0 {
		parts = append(parts, "primary_topic.id:"+strings.Join(topics, "|"))
	}
	dateFrom := cfg.Filters.DateFrom
	if dateFrom == "" {
		dateFrom = "2003-01-01"
	}
	dateTo := cfg.Filters.DateTo
	if dateTo == "" {
		dateTo = "2024-12-31"
	}
	parts = append(parts, "from_publication_date:"+dateFrom)
	parts = append(parts, "to_publication_date:"+dateTo)
	if len(cfg.Filters.DocTypes) > 0 {
		parts = append(parts, "type:"+strings.Join(cfg.Filters.DocTypes, "|"))
	}
	filter := strings.Join(parts, ",")

	var allGroups []openalex.GroupByItem
	cursor := "*"
	for {
		resp, err := client.FetchGroupBy(ctx, filter, "primary_topic.id", cursor)
		if err != nil {
			errStr := "OpenAlex request failed: " + err.Error()
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: errStr}},
				IsError: true,
			}, GetTopicsResult{Markdown: ""}, nil
		}
		if resp == nil || len(resp.GroupBy) == 0 {
			break
		}
		allGroups = append(allGroups, resp.GroupBy...)
		if resp.Meta.NextCursor == "" || resp.Meta.NextCursor == cursor {
			break
		}
		cursor = resp.Meta.NextCursor
	}

	if len(allGroups) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: "No topics found matching the current keyword configurations."}},
		}, GetTopicsResult{Markdown: "No topics found."}, nil
	}

	sort.Slice(allGroups, func(i, j int) bool {
		return allGroups[i].Count > allGroups[j].Count
	})

	var totalPapers int
	for _, g := range allGroups {
		totalPapers += g.Count
	}

	type EnrichedTopic struct {
		TopicID     string
		DisplayName string
		Description string
		Count       int
		Percentage  float64
	}

	enriched := make([]EnrichedTopic, len(allGroups))
	var wg sync.WaitGroup

	for i, g := range allGroups {
		wg.Add(1)
		go func(idx int, item openalex.GroupByItem) {
			defer wg.Done()

			percentage := 0.0
			if totalPapers > 0 {
				percentage = float64(item.Count) / float64(totalPapers) * 100
			}

			topicID := item.Key
			if lastSlash := strings.LastIndex(topicID, "/"); lastSlash != -1 {
				topicID = topicID[lastSlash+1:]
			}

			eTopic := EnrichedTopic{
				TopicID:     topicID,
				DisplayName: item.KeyDisplayName,
				Count:       item.Count,
				Percentage:  percentage,
			}

			if args.Details {
				details, err := client.FetchTopicDetails(ctx, topicID)
				if err == nil && details != nil {
					if details.DisplayName != "" {
						eTopic.DisplayName = details.DisplayName
					}
					eTopic.Description = details.Description
				}
			}

			enriched[idx] = eTopic
		}(i, g)
	}
	wg.Wait()

	var md strings.Builder
	md.WriteString(fmt.Sprintf("## Topics found in search results (%d topics, %d papers total)\n\n", len(enriched), totalPapers))
	md.WriteString("| Topic ID | Topic Name | Description | Paper Count | Percentage |\n")
	md.WriteString("| :--- | :--- | :--- | :---: | :---: |\n")

	for _, t := range enriched {
		desc := t.Description
		if len(desc) > 80 {
			desc = desc[:77] + "..."
		}
		if desc == "" {
			desc = "—"
		}
		md.WriteString(fmt.Sprintf("| `%s` | %s | %s | %d | %.2f%% |\n", t.TopicID, t.DisplayName, desc, t.Count, t.Percentage))
	}

	markdownStr := md.String()

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: markdownStr}},
	}, GetTopicsResult{Markdown: markdownStr}, nil
}

// RegisterMCPResources registers all static and dynamic resources.
func (s *APIServer) RegisterMCPResources() error {
	kt := &mcp.ResourceTemplate{
		URITemplate: "stratum://knowledge/{category}/{name}",
		Name:        "Stratum Knowledge Base",
		Description: "Static operating manuals, methodology references, agent SOPs, and checklists.",
		MIMEType:    "application/json",
	}
	s.mcpServer.AddResourceTemplate(kt, s.handleReadKnowledgeResource())

	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "stratum://state/project",
		Name:        "Current Project Summary",
		Description: "Provides a high-level summary of the active project (name, papers loaded, downloaded status, stage actions).",
		MIMEType:    "application/json",
	}, s.handleReadStateProject())

	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "stratum://state/workflow",
		Name:        "Current Workflow State",
		Description: "Provides active pipeline execution status (running status, list of completed and pending tasks).",
		MIMEType:    "application/json",
	}, s.handleReadStateWorkflow())

	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "stratum://state/workflow/next",
		Name:        "Recommended Next Action",
		Description: "Exposes planner-driven advice recommending the next action step for the agent.",
		MIMEType:    "text/plain",
	}, s.handleReadStateNext())

	s.mcpServer.AddResource(&mcp.Resource{
		URI:         "stratum://state/history",
		Name:        "Project Configuration History",
		Description: "Retrieves the SQLite database config history log showing previous runs.",
		MIMEType:    "application/json",
	}, s.handleReadStateHistory())

	return nil
}

// RunMCPStdio runs the integrated MCP server over the standard input/output transport.
func (s *APIServer) RunMCPStdio(ctx context.Context) error {
	log.SetOutput(os.Stderr)
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}

func (s *APIServer) handleReadKnowledgeResource() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		if !strings.HasPrefix(uri, "stratum://knowledge/") {
			return nil, fmt.Errorf("invalid resource URI: %s", uri)
		}

		pathPart := strings.TrimPrefix(uri, "stratum://knowledge/")
		parts := strings.Split(pathPart, "/")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid knowledge path: %s", pathPart)
		}

		category := parts[0]
		name := parts[1]

		filePath := fmt.Sprintf("%s/%s.md", category, name)
		data, err := docs.DocsFS.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("knowledge document not found: %s (%v)", filePath, err)
		}

		markdownText := string(data)
		var frontmatter string
		body := markdownText

		if strings.HasPrefix(markdownText, "---") {
			parts := strings.SplitN(markdownText, "---", 3)
			if len(parts) == 3 {
				frontmatter = parts[1]
				body = parts[2]
			}
		}

		metadataMap := make(map[string]interface{})
		metadataMap["markdown"] = body
		metadataMap["uri"] = uri

		lines := strings.Split(frontmatter, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || !strings.Contains(line, ":") {
				continue
			}
			kv := strings.SplitN(line, ":", 2)
			k := strings.TrimSpace(kv[0])
			v := strings.TrimSpace(kv[1])

			if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") {
				v = strings.Trim(v, "[]")
				listParts := strings.Split(v, ",")
				var cleanList []string
				for _, lp := range listParts {
					cleanList = append(cleanList, strings.TrimSpace(lp))
				}
				metadataMap[k] = cleanList
			} else {
				metadataMap[k] = v
			}
		}

		jsonData, err := json.MarshalIndent(metadataMap, "", "  ")
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}
}

func (s *APIServer) handleReadStateProject() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		project := "default"
		parsedURI, err := url.Parse(req.Params.URI)
		if err == nil {
			projParam := parsedURI.Query().Get("project")
			if projParam != "" {
				project = projParam
			}
		}

		configDBPath, papersDBPath, _, _, _ := s.getProjectPaths(project)

		cfg, err := config.LoadConfig(configDBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration for project %q: %v", project, err)
		}

		var dbConn *sql.DB
		if _, err := os.Stat(papersDBPath); err == nil {
			dbConn, _ = sql.Open("duckdb", papersDBPath)
			if dbConn != nil {
				defer dbConn.Close()
			}
		}

		summary, _, _, err := workflow.AnalyzeState(cfg, dbConn)
		if err != nil {
			return nil, err
		}

		summary.Project = project

		jsonData, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}
}

func (s *APIServer) handleReadStateWorkflow() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		project := "default"
		parsedURI, err := url.Parse(req.Params.URI)
		if err == nil {
			projParam := parsedURI.Query().Get("project")
			if projParam != "" {
				project = projParam
			}
		}

		configDBPath, papersDBPath, _, _, _ := s.getProjectPaths(project)

		cfg, err := config.LoadConfig(configDBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration for project %q: %v", project, err)
		}

		var dbConn *sql.DB
		if _, err := os.Stat(papersDBPath); err == nil {
			dbConn, _ = sql.Open("duckdb", papersDBPath)
			if dbConn != nil {
				defer dbConn.Close()
			}
		}

		_, state, _, err := workflow.AnalyzeState(cfg, dbConn)
		if err != nil {
			return nil, err
		}

		s.mu.Lock()
		status, exists := s.pipelineStatuses[project]
		if exists {
			state.Running = status.Syncing
		}
		s.mu.Unlock()

		jsonData, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}
}

func (s *APIServer) handleReadStateNext() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		project := "default"
		parsedURI, err := url.Parse(req.Params.URI)
		if err == nil {
			projParam := parsedURI.Query().Get("project")
			if projParam != "" {
				project = projParam
			}
		}

		configDBPath, papersDBPath, _, _, _ := s.getProjectPaths(project)

		cfg, err := config.LoadConfig(configDBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration for project %q: %v", project, err)
		}

		var dbConn *sql.DB
		if _, err := os.Stat(papersDBPath); err == nil {
			dbConn, _ = sql.Open("duckdb", papersDBPath)
			if dbConn != nil {
				defer dbConn.Close()
			}
		}

		_, _, nextAction, err := workflow.AnalyzeState(cfg, dbConn)
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "text/plain",
					Text:     nextAction,
				},
			},
		}, nil
	}
}

func (s *APIServer) handleReadStateHistory() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		project := "default"
		parsedURI, err := url.Parse(req.Params.URI)
		if err == nil {
			projParam := parsedURI.Query().Get("project")
			if projParam != "" {
				project = projParam
			}
		}

		configDBPath, _, _, _, _ := s.getProjectPaths(project)

		dbConn, err := sql.Open("duckdb", configDBPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open config database: %v", err)
		}
		defer dbConn.Close()

		rows, err := dbConn.Query("SELECT version, timestamp, label, keywords, topics, anchors FROM config_history ORDER BY version DESC")
		if err != nil {
			jsonData, _ := json.MarshalIndent([]interface{}{}, "", "  ")
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{
					{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(jsonData),
					},
				},
			}, nil
		}
		defer rows.Close()

		type HistoryItem struct {
			Version   int    `json:"version"`
			Timestamp string `json:"timestamp"`
			Label     string `json:"label"`
			Keywords  string `json:"keywords"`
			Topics    string `json:"topics"`
			Anchors   string `json:"anchors"`
		}

		var history []HistoryItem
		for rows.Next() {
			var item HistoryItem
			if err := rows.Scan(&item.Version, &item.Timestamp, &item.Label, &item.Keywords, &item.Topics, &item.Anchors); err == nil {
				history = append(history, item)
			}
		}

		jsonData, err := json.MarshalIndent(history, "", "  ")
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      req.Params.URI,
					MIMEType: "application/json",
					Text:     string(jsonData),
				},
			},
		}, nil
	}
}
