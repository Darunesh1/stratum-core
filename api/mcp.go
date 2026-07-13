package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"stratum/config"
	"stratum/docs"
	"stratum/impute"
	"stratum/openalex"
	"stratum/tfidf"
	"stratum/wos"
	"stratum/workflow"
)

// MCP Types mapped from standalone server
type ValidateArgs struct {
	Project string `json:"project,omitempty" jsonschema:"Optional project name. Defaults to active project"`
}

type ValidateResult struct {
	Valid  bool     `json:"valid" jsonschema:"Indicates if keywords and topics are structurally valid"`
	Errors []string `json:"errors" jsonschema:"List of validation error messages"`
}

type SearchArgs struct {
	Project      string `json:"project,omitempty" jsonschema:"Optional project name. Defaults to active project"`
	CheckAnchors bool   `json:"check_anchors,omitempty" jsonschema:"Optional flag to check anchor DOI coverage"`
}

type SearchResult struct {
	TotalCount     int      `json:"total_count" jsonschema:"Total matching papers"`
	AnchorsTotal   int      `json:"anchors_total,omitempty"`
	AnchorsMatched int      `json:"anchors_matched,omitempty"`
	AnchorsMissing []string `json:"anchors_missing,omitempty"`
	RetrievalNote  string   `json:"retrieval_note,omitempty"`
}

type DownloadArgs struct {
	Project     string `json:"project,omitempty" jsonschema:"Optional project name. Defaults to active project"`
	OutputJSONL string `json:"output_jsonl,omitempty" jsonschema:"Optional path to write downloaded JSONL"`
}

type DownloadResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ConvertDBArgs struct {
	Project   string `json:"project,omitempty" jsonschema:"Optional project name. Defaults to active project"`
	JSONLPath string `json:"jsonl_path,omitempty" jsonschema:"Optional path to input JSONL"`
}

type ConvertDBResult struct {
	Status        string `json:"status"`
	PapersLoaded  int    `json:"papers_loaded"`
	AuthorsLoaded int    `json:"authors_loaded"`
	InstsLoaded   int    `json:"institutions_loaded"`
	Errors        int    `json:"errors"`
}

type ImputeArgs struct {
	Project  string `json:"project,omitempty" jsonschema:"Optional project name. Defaults to active project"`
	Pipeline string `json:"pipeline,omitempty" jsonschema:"Pipeline stage to execute: crossref, llm, pdf, or all"`
	Limit    int    `json:"limit,omitempty" jsonschema:"Limit for PDF extraction"`
}

type ImputeResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type GetTopicsArgs struct {
	Project string `json:"project,omitempty" jsonschema:"Optional project name. Defaults to active project"`
	Details bool   `json:"details,omitempty"`
}

type GetTopicsResult struct {
	Markdown string `json:"markdown"`
}

// Project Management structures
type CreateProjectArgs struct {
	Name string `json:"name" jsonschema:"The unique alphanumeric name of the research project to create"`
}

type CreateProjectResult struct {
	Status string `json:"status"`
	Name   string `json:"name"`
}

type ListProjectsArgs struct{}

type ListProjectsResult struct {
	Projects []string `json:"projects"`
}

type SelectProjectArgs struct {
	Project string `json:"project" jsonschema:"The project name to set as active for subsequent tool invocations"`
}

type SelectProjectResult struct {
	Status string `json:"status"`
	Active string `json:"active"`
}

type GetProjectConfigArgs struct {
	Project        string `json:"project,omitempty" jsonschema:"Optional project name. Defaults to active project"`
	IncludeQuery   bool   `json:"include_query,omitempty" jsonschema:"Set true to explicitly retrieve the full boolean query keywords"`
	IncludeTopics  bool   `json:"include_topics,omitempty" jsonschema:"Set true to explicitly retrieve the full list of topic IDs"`
	IncludeAnchors bool   `json:"include_anchors,omitempty" jsonschema:"Set true to explicitly retrieve the full list of anchor DOIs"`
}

type GetProjectConfigResult struct {
	Config        config.AppConfig `json:"config"`
	Keywords      string           `json:"keywords,omitempty"`
	Topics        string           `json:"topics,omitempty"`
	Anchors       string           `json:"anchors,omitempty"`
	KeywordsLen   int              `json:"keywords_length"`
	TopicsCount   int              `json:"topics_count"`
	AnchorsCount  int              `json:"anchors_count"`
	RetrievalNote string           `json:"retrieval_note,omitempty"`
}

type UpdateProjectConfigArgs struct {
	Project   string   `json:"project,omitempty" jsonschema:"Optional project name"`
	Keywords  string   `json:"keywords,omitempty" jsonschema:"Optional boolean search keywords query"`
	Topics    []string `json:"topics,omitempty" jsonschema:"Optional list of OpenAlex topic IDs"`
	Anchors   []string `json:"anchors,omitempty" jsonschema:"Optional list of anchor DOIs"`
	DateFrom  string   `json:"date_from,omitempty" jsonschema:"Optional start date (YYYY-MM-DD)"`
	DateTo    string   `json:"date_to,omitempty" jsonschema:"Optional end date (YYYY-MM-DD)"`
	DocTypes  []string `json:"doc_types,omitempty" jsonschema:"Optional list of document types"`
	Label     string   `json:"label,omitempty" jsonschema:"Label description for this version revision"`
}

type UpdateProjectConfigResult struct {
	Status string `json:"status"`
}

// Anchor Prep structures
type UploadReferenceArgs struct {
	FilePath string `json:"file_path" jsonschema:"Absolute path of reference file in local workspace"`
	Project  string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type UploadReferenceResult struct {
	Status   string   `json:"status"`
	Filename string   `json:"filename"`
	Columns  []string `json:"columns"`
	RowCount int      `json:"row_count"`
}

type ExtractQueryAndAnchorsArgs struct {
	Filename       string `json:"filename" jsonschema:"Filename of uploaded reference sheet inside project uploads folder"`
	TitleColumn    string `json:"title_column" jsonschema:"Column name containing paper titles"`
	AbstractColumn string `json:"abstract_column" jsonschema:"Column name containing paper abstracts"`
	DOIColumn      string `json:"doi_column,omitempty" jsonschema:"Column name containing paper DOIs"`
	SaveToConfig   bool   `json:"save_to_config,omitempty" jsonschema:"Save suggested keywords and extracted anchors directly into project config"`
	Project        string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type ExtractQueryAndAnchorsResult struct {
	Keywords        string   `json:"keywords"`
	ExtractedDOIs   []string `json:"extracted_dois"`
	AnchorsSaved    int      `json:"anchors_saved"`
	UnindexedReview []string `json:"unindexed_review,omitempty"`
}

type ValidateAnchorsArgs struct {
	Project string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type AnchorStatus struct {
	DOI          string `json:"doi"`
	Status       string `json:"status"`
	IndexDetails string `json:"details"`
}

type ValidateAnchorsResult struct {
	Total         int            `json:"total"`
	IndexedCount  int            `json:"indexed_count"`
	MissingCount  int            `json:"missing_count"`
	AnchorsReport []AnchorStatus `json:"anchors_report"`
}

// Search & sample structures
type GetSampleArgs struct {
	Size    int    `json:"size,omitempty" jsonschema:"Number of records to fetch. Default 20, max 200"`
	Project string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type GetSampleResult struct {
	TotalMatches int             `json:"total_matches"`
	Samples      []openalex.Work `json:"samples"`
}

// Exploration structures
type QuerySQLArgs struct {
	Query   string `json:"query" jsonschema:"Factual SELECT statement to query collected data (read-only)"`
	Project string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type QuerySQLResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

type GetStatisticsArgs struct {
	Project string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type GetStatisticsResult struct {
	TotalPapers            int     `json:"total_papers"`
	TotalAuthors           int     `json:"total_authors"`
	TotalInstitutions      int     `json:"total_institutions"`
	TotalCountries         int     `json:"total_countries"`
	MissingCountryCount    int     `json:"missing_country_count"`
	MissingInstitutionID   int     `json:"missing_institution_id_count"`
	ImputedCountryCount    int     `json:"imputed_country_count"`
	ImputationCompleteness float64 `json:"imputation_completeness_percent"`
}

// WoS Integration structures
type UploadWoSArgs struct {
	FilePath string `json:"file_path" jsonschema:"Absolute path of WoS file in local workspace"`
	Project  string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type UploadWoSResult struct {
	Status   string `json:"status"`
	Filename string `json:"filename"`
	Size     int64  `json:"size_bytes"`
}

type ImportWoSDoisArgs struct {
	Filename string `json:"filename" jsonschema:"Filename of WoS upload inside project uploads folder"`
	Project  string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type ImportWoSDoisResult struct {
	Status          string `json:"status"`
	ImportedRecords int    `json:"imported_records"`
}

type SyncWoSOpenAlexArgs struct {
	Filename string `json:"filename" jsonschema:"Filename of WoS upload inside project uploads folder"`
	Project  string `json:"project,omitempty" jsonschema:"Optional project name"`
}

type SyncWoSOpenAlexResult struct {
	TotalWoS          int     `json:"total_wos"`
	TotalDB           int     `json:"total_db"`
	ExactDOIMatches   int     `json:"exact_doi_matches"`
	FuzzyTitleMatches int     `json:"fuzzy_title_matches"`
	OverlapPercentage float64 `json:"overlap_percentage"`
}

func (s *APIServer) resolveProjectFromConfigPath(configPath string) string {
	if configPath == "" {
		return s.currentProject
	}
	cleaned := filepath.ToSlash(filepath.Clean(configPath))
	parts := strings.Split(cleaned, "/")
	for i := 0; i < len(parts)-2; i++ {
		if parts[i] == "projects" {
			return sanitizeProjectName(parts[i+1])
		}
	}
	return s.currentProject
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

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "create_project",
		Description: "Create a new isolated research project directory and setup default config database.",
	}, s.handleCreateProjectMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "list_projects",
		Description: "List all existing research project workspace folders.",
	}, s.handleListProjectsMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "select_project",
		Description: "Set the active research project for subsequent tool invocations.",
	}, s.handleSelectProjectMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_project_config",
		Description: "Retrieve active configuration settings (keywords, anchors, topics, metadata filters) for a project.",
	}, s.handleGetProjectConfigMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "update_project_config",
		Description: "Update keywords, topic IDs, anchor DOIs, publication years, or document types in the project config database.",
	}, s.handleUpdateProjectConfigMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "upload_reference_file",
		Description: "Upload reference publications (CSV/Excel) containing anchor DOIs into the project uploads folder.",
	}, s.handleUploadReferenceMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "extract_query_and_anchors",
		Description: "Apply TF-IDF extraction to uploaded files to discover keywords and extract anchor publication DOIs.",
	}, s.handleExtractQueryAndAnchorsMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "validate_anchors",
		Description: "Verify recall index coverage of configured anchor DOIs on the OpenAlex API.",
	}, s.handleValidateAnchorsMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_sample",
		Description: "Fetch a random sample of academic papers matching current configuration filters from OpenAlex for review.",
	}, s.handleGetSampleMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "query_sql",
		Description: "Execute custom read-only DuckDB SELECT queries against the local database of downloaded papers.",
	}, s.handleQuerySQLMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "get_statistics",
		Description: "Retrieve aggregate counts (total papers, unique authors, unique countries) and metadata completeness status from the local database.",
	}, s.handleGetStatisticsMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "upload_wos_file",
		Description: "Upload a Web of Science export file (CSV, Excel, or Plain Text) to the project uploads directory.",
	}, s.handleUploadWoSMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "import_wos_dois",
		Description: "Parse the uploaded Web of Science file and import its DOIs/records into the project database.",
	}, s.handleImportWoSDoisMCP)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "sync_wos_openalex",
		Description: "Compare DOIs in the imported WoS files against the OpenAlex database to evaluate paper overlap and coverage.",
	}, s.handleSyncWoSOpenAlexMCP)

	return nil
}

func (s *APIServer) handleValidate(ctx context.Context, req *mcp.CallToolRequest, args ValidateArgs) (*mcp.CallToolResult, ValidateResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}
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
	project := args.Project
	if project == "" {
		project = s.currentProject
	}
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
	var displayMissing []string
	var retrievalNote string
	if args.CheckAnchors {
		text += fmt.Sprintf(" Anchor match coverage: %d/%d matches.", matchedCount, len(anchors))
		if len(missingDOIs) > 10 {
			displayMissing = missingDOIs[:10]
			retrievalNote = fmt.Sprintf("Only the first 10 missing DOIs are shown to keep context size low (%d missing total). " +
				"Use the 'validate_anchors' tool to check the full list of missing anchors and verify their indexing status.", len(missingDOIs))
		} else {
			displayMissing = missingDOIs
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}, SearchResult{
		TotalCount:     count,
		AnchorsTotal:   len(anchors),
		AnchorsMatched: matchedCount,
		AnchorsMissing: displayMissing,
		RetrievalNote:  retrievalNote,
	}, nil
}

func (s *APIServer) handleDownload(ctx context.Context, req *mcp.CallToolRequest, args DownloadArgs) (*mcp.CallToolResult, DownloadResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}
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
	project := args.Project
	if project == "" {
		project = s.currentProject
	}
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
	project := args.Project
	if project == "" {
		project = s.currentProject
	}
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
	project := args.Project
	if project == "" {
		project = s.currentProject
	}
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

// Project Management handlers
func (s *APIServer) handleCreateProjectMCP(ctx context.Context, req *mcp.CallToolRequest, args CreateProjectArgs) (*mcp.CallToolResult, CreateProjectResult, error) {
	name := sanitizeProjectName(args.Name)
	if name == "" || name == "default" {
		return nil, CreateProjectResult{}, fmt.Errorf("invalid project name")
	}

	if err := s.ensureProjectDirs(name); err != nil {
		return nil, CreateProjectResult{}, fmt.Errorf("failed to create project directories: %w", err)
	}

	configDB, err := s.getConfigDB(name)
	if err != nil {
		return nil, CreateProjectResult{}, fmt.Errorf("failed to initialize config DB: %w", err)
	}

	configDBPath, _, _, _, _ := s.getProjectPaths(name)
	cfg, err := config.LoadConfig(configDBPath)
	var keywords, topics, anchors string
	if err == nil {
		keywords = cfg.Keywords
		topics = strings.Join(cfg.Topics, "\n")
		anchors = strings.Join(cfg.Anchors, "\n")
	}
	_ = s.appendConfigRevision(configDB, keywords, topics, anchors, "Project Created")

	return &mcp.CallToolResult{}, CreateProjectResult{
		Status: "success",
		Name:   name,
	}, nil
}

func (s *APIServer) handleListProjectsMCP(ctx context.Context, req *mcp.CallToolRequest, args ListProjectsArgs) (*mcp.CallToolResult, ListProjectsResult, error) {
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

	return &mcp.CallToolResult{}, ListProjectsResult{
		Projects: projects,
	}, nil
}

func (s *APIServer) handleSelectProjectMCP(ctx context.Context, req *mcp.CallToolRequest, args SelectProjectArgs) (*mcp.CallToolResult, SelectProjectResult, error) {
	name := sanitizeProjectName(args.Project)
	if name == "" {
		return nil, SelectProjectResult{}, fmt.Errorf("invalid project name")
	}
	if name != "default" {
		if fi, err := os.Stat(filepath.Join("projects", name)); err != nil || !fi.IsDir() {
			return nil, SelectProjectResult{}, fmt.Errorf("project %q does not exist. Call create_project first", name)
		}
	}

	s.mu.Lock()
	s.currentProject = name
	s.mu.Unlock()

	return &mcp.CallToolResult{}, SelectProjectResult{
		Status: "success",
		Active: name,
	}, nil
}

func (s *APIServer) handleGetProjectConfigMCP(ctx context.Context, req *mcp.CallToolRequest, args GetProjectConfigArgs) (*mcp.CallToolResult, GetProjectConfigResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	configDBPath, _, _, _, _ := s.getProjectPaths(project)
	s.ensureProjectDirs(project)

	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		return nil, GetProjectConfigResult{}, fmt.Errorf("failed to load config: %w", err)
	}

	keywordsCount := len(cfg.Keywords)
	topicsCount := len(cfg.Topics)
	anchorsCount := len(cfg.Anchors)

	var keywords string
	var topics string
	var anchors string

	if args.IncludeQuery {
		keywords = cfg.Keywords
	} else {
		cfg.Keywords = ""
	}

	if args.IncludeTopics {
		topics = strings.Join(cfg.Topics, "\n")
	} else {
		cfg.Topics = nil
	}

	if args.IncludeAnchors {
		anchors = strings.Join(cfg.Anchors, "\n")
	} else {
		cfg.Anchors = nil
	}

	retrievalNote := "Large query string/keywords, topic IDs, and anchor DOIs are hidden by default to keep context size low. " +
		"If you need them, call get_project_config with include_query: true, include_topics: true, or include_anchors: true."

	return &mcp.CallToolResult{}, GetProjectConfigResult{
		Config:        *cfg,
		Keywords:      keywords,
		Topics:        topics,
		Anchors:       anchors,
		KeywordsLen:   keywordsCount,
		TopicsCount:   topicsCount,
		AnchorsCount:  anchorsCount,
		RetrievalNote: retrievalNote,
	}, nil
}

func (s *APIServer) handleUpdateProjectConfigMCP(ctx context.Context, req *mcp.CallToolRequest, args UpdateProjectConfigArgs) (*mcp.CallToolResult, UpdateProjectConfigResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	configDBPath, _, jsonlDir, dbDir, _ := s.getProjectPaths(project)
	s.ensureProjectDirs(project)

	configDB, err := s.getConfigDB(project)
	if err != nil {
		return nil, UpdateProjectConfigResult{}, fmt.Errorf("failed to connect to config DB: %w", err)
	}

	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		return nil, UpdateProjectConfigResult{}, fmt.Errorf("failed to load config: %w", err)
	}

	if args.Keywords != "" {
		errs := openalex.ValidateKeywords(args.Keywords)
		if len(errs) > 0 {
			return nil, UpdateProjectConfigResult{}, fmt.Errorf("strict keywords validation failed: %s", strings.Join(errs, "; "))
		}
		cfg.Keywords = args.Keywords
	}

	if len(args.Topics) > 0 {
		cfg.Topics = args.Topics
	}

	if len(args.Anchors) > 0 {
		if len(args.Anchors) > 385 {
			cfg.Anchors = args.Anchors[:385]
		} else {
			cfg.Anchors = args.Anchors
		}
	}

	if args.DateFrom != "" {
		cfg.Filters.DateFrom = args.DateFrom
	}
	if args.DateTo != "" {
		cfg.Filters.DateTo = args.DateTo
	}
	if len(args.DocTypes) > 0 {
		cfg.Filters.DocTypes = args.DocTypes
	}

	cfg.Output.JSONLDir = jsonlDir
	cfg.Output.DBDir = dbDir

	if err := config.SaveConfig(configDBPath, cfg); err != nil {
		return nil, UpdateProjectConfigResult{}, fmt.Errorf("failed to save config to DB: %w", err)
	}

	topicsStr := strings.Join(cfg.Topics, "\n")
	anchorsStr := strings.Join(cfg.Anchors, "\n")
	label := args.Label
	if label == "" {
		label = "MCP Config Update"
	}
	_ = s.appendConfigRevision(configDB, cfg.Keywords, topicsStr, anchorsStr, label)

	return &mcp.CallToolResult{}, UpdateProjectConfigResult{Status: "success"}, nil
}

// Anchor & Reference handlers
func (s *APIServer) handleUploadReferenceMCP(ctx context.Context, req *mcp.CallToolRequest, args UploadReferenceArgs) (*mcp.CallToolResult, UploadReferenceResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	if args.FilePath == "" {
		return nil, UploadReferenceResult{}, fmt.Errorf("file_path parameter is required")
	}

	ext := strings.ToLower(filepath.Ext(args.FilePath))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" {
		return nil, UploadReferenceResult{}, fmt.Errorf("unsupported file format %q. Please upload a .csv, .xlsx, or .xls file", ext)
	}

	_, _, _, _, uploadDir := s.getProjectPaths(project)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, UploadReferenceResult{}, fmt.Errorf("failed to create uploads directory: %w", err)
	}

	srcFile, err := os.Open(args.FilePath)
	if err != nil {
		return nil, UploadReferenceResult{}, fmt.Errorf("failed to open source reference file: %w", err)
	}
	defer srcFile.Close()

	safeName := fmt.Sprintf("upload_%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join(uploadDir, safeName)
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, UploadReferenceResult{}, fmt.Errorf("failed to create destination file in uploads: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, srcFile); err != nil {
		return nil, UploadReferenceResult{}, fmt.Errorf("failed to write file to uploads directory: %w", err)
	}

	var headers []string
	var rowCount int
	if ext == ".csv" {
		headers, rowCount, err = parseCSVHeadersAndCount(destPath)
	} else {
		headers, rowCount, err = parseExcelHeadersAndCount(destPath)
	}

	if err != nil {
		return nil, UploadReferenceResult{}, fmt.Errorf("failed to parse uploaded file: %w", err)
	}

	return &mcp.CallToolResult{}, UploadReferenceResult{
		Status:   "success",
		Filename: safeName,
		Columns:  headers,
		RowCount: rowCount,
	}, nil
}

func (s *APIServer) handleExtractQueryAndAnchorsMCP(ctx context.Context, req *mcp.CallToolRequest, args ExtractQueryAndAnchorsArgs) (*mcp.CallToolResult, ExtractQueryAndAnchorsResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	if args.Filename == "" {
		return nil, ExtractQueryAndAnchorsResult{}, fmt.Errorf("filename is required")
	}
	if args.TitleColumn == "" || args.AbstractColumn == "" {
		return nil, ExtractQueryAndAnchorsResult{}, fmt.Errorf("title_column and abstract_column are required")
	}

	configDBPath, _, _, _, uploadsDir := s.getProjectPaths(project)
	filePath := filepath.Join(uploadsDir, args.Filename)
	ext := strings.ToLower(filepath.Ext(filePath))

	var docs []string
	var err error
	if ext == ".csv" {
		docs, err = loadCSVDocuments(filePath, args.TitleColumn, args.AbstractColumn)
	} else {
		docs, err = loadExcelDocuments(filePath, args.TitleColumn, args.AbstractColumn)
	}

	if err != nil {
		return nil, ExtractQueryAndAnchorsResult{}, fmt.Errorf("failed to load documents: %w", err)
	}

	var dois []string
	if args.DOIColumn != "" {
		if ext == ".csv" {
			dois, err = extractDOIsFromCSV(filePath, args.DOIColumn)
		} else {
			dois, err = extractDOIsFromExcel(filePath, args.DOIColumn)
		}
		if err != nil {
			return nil, ExtractQueryAndAnchorsResult{}, fmt.Errorf("failed to extract DOIs: %w", err)
		}

		if len(dois) > 0 && args.SaveToConfig {
			if len(dois) > 385 {
				dois = dois[:385]
			}
			cfg, err := config.LoadConfig(configDBPath)
			if err == nil {
				cfg.Anchors = dois
				_ = config.SaveConfig(configDBPath, cfg)
			}
		}
	}

	keywords := tfidf.ExtractKeywords(docs, 2, 3, 2, 0.85, 50)
	var termList []string
	for _, term := range keywords {
		termList = append(termList, term.Term)
	}
	suggestedKeywords := strings.Join(termList, " OR ")

	if args.SaveToConfig && suggestedKeywords != "" {
		cfg, err := config.LoadConfig(configDBPath)
		if err == nil {
			cfg.Keywords = suggestedKeywords
			_ = config.SaveConfig(configDBPath, cfg)
		}
	}

	return &mcp.CallToolResult{}, ExtractQueryAndAnchorsResult{
		Keywords:      suggestedKeywords,
		ExtractedDOIs: dois,
		AnchorsSaved:  len(dois),
	}, nil
}

func (s *APIServer) handleValidateAnchorsMCP(ctx context.Context, req *mcp.CallToolRequest, args ValidateAnchorsArgs) (*mcp.CallToolResult, ValidateAnchorsResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	configDBPath, _, _, _, _ := s.getProjectPaths(project)
	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		return nil, ValidateAnchorsResult{}, fmt.Errorf("failed to load configuration: %w", err)
	}

	if len(cfg.Anchors) == 0 {
		return &mcp.CallToolResult{}, ValidateAnchorsResult{
			Total:        0,
			IndexedCount: 0,
		}, nil
	}

	client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, 200, 5, 3, 1)
	var report []AnchorStatus
	var indexed int
	var missingCount int

	for _, doi := range cfg.Anchors {
		doi = strings.TrimSpace(doi)
		if doi == "" {
			continue
		}
		count, err := client.GetTotalCount(ctx, "doi:"+doi)
		if err == nil && count > 0 {
			indexed++
			report = append(report, AnchorStatus{
				DOI:          doi,
				Status:       "indexed",
				IndexDetails: "Found on OpenAlex API",
			})
		} else {
			missingCount++
			report = append(report, AnchorStatus{
				DOI:          doi,
				Status:       "missing",
				IndexDetails: "DOI not found on OpenAlex search endpoints (either not indexed or typo in DOI string)",
			})
		}
	}

	return &mcp.CallToolResult{}, ValidateAnchorsResult{
		Total:         len(cfg.Anchors),
		IndexedCount:  indexed,
		MissingCount:  missingCount,
		AnchorsReport: report,
	}, nil
}

// Search & sample handlers
func (s *APIServer) handleGetSampleMCP(ctx context.Context, req *mcp.CallToolRequest, args GetSampleArgs) (*mcp.CallToolResult, GetSampleResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	configDBPath, _, _, _, _ := s.getProjectPaths(project)
	cfg, err := config.LoadConfig(configDBPath)
	if err != nil {
		return nil, GetSampleResult{}, fmt.Errorf("failed to load config: %w", err)
	}

	size := args.Size
	if size <= 0 {
		size = 20
	}
	if size > 200 {
		size = 200
	}

	client := openalex.NewClient(cfg.API.Keys, cfg.API.Email, 200, 5, 3, 1)

	parts := []string{"title_and_abstract.search:" + cfg.Keywords}
	if len(cfg.Topics) > 0 {
		parts = append(parts, "primary_topic.id:"+strings.Join(cfg.Topics, "|"))
	}
	parts = append(parts, "from_publication_date:"+cfg.Filters.DateFrom)
	parts = append(parts, "to_publication_date:"+cfg.Filters.DateTo)
	if len(cfg.Filters.DocTypes) > 0 {
		parts = append(parts, "type:"+strings.Join(cfg.Filters.DocTypes, "|"))
	}
	apiFilter := strings.Join(parts, ",")

	count, err := client.GetTotalCount(ctx, apiFilter)
	if err != nil {
		count = 0
	}

	samples, err := client.FetchSample(ctx, apiFilter, size, 42)
	if err != nil {
		return nil, GetSampleResult{}, fmt.Errorf("failed to fetch sample from OpenAlex: %w", err)
	}

	return &mcp.CallToolResult{}, GetSampleResult{
		TotalMatches: count,
		Samples:      samples,
	}, nil
}

// Exploration handlers
func (s *APIServer) handleQuerySQLMCP(ctx context.Context, req *mcp.CallToolRequest, args QuerySQLArgs) (*mcp.CallToolResult, QuerySQLResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	q := strings.TrimSpace(args.Query)
	if q == "" {
		return nil, QuerySQLResult{}, fmt.Errorf("query is empty")
	}

	upperQ := strings.ToUpper(q)
	if !strings.HasPrefix(upperQ, "SELECT") {
		return nil, QuerySQLResult{}, fmt.Errorf("only read-only SELECT queries are allowed via MCP")
	}

	badKeywords := []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "REPLACE", "TRUNCATE"}
	for _, kw := range badKeywords {
		if strings.Contains(upperQ, kw) {
			return nil, QuerySQLResult{}, fmt.Errorf("unauthorized SQL keyword %q detected in query", kw)
		}
	}

	_, _, _, dbDir, _ := s.getProjectPaths(project)
	dbPath := filepath.Join(dbDir, "papers.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, QuerySQLResult{}, fmt.Errorf("duckdb database for project %q does not exist. Call convert_db first", project)
	}

	dbConn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, QuerySQLResult{}, fmt.Errorf("failed to open database: %w", err)
	}
	defer dbConn.Close()

	rows, err := dbConn.QueryContext(ctx, q)
	if err != nil {
		return nil, QuerySQLResult{}, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, QuerySQLResult{}, fmt.Errorf("failed to fetch column names: %w", err)
	}

	var results []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, QuerySQLResult{}, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columns[i]
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	return &mcp.CallToolResult{}, QuerySQLResult{
		Columns: cols,
		Rows:    results,
	}, nil
}

func (s *APIServer) handleGetStatisticsMCP(ctx context.Context, req *mcp.CallToolRequest, args GetStatisticsArgs) (*mcp.CallToolResult, GetStatisticsResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	_, _, _, dbDir, _ := s.getProjectPaths(project)
	dbPath := filepath.Join(dbDir, "papers.db")
	if _, err := os.Stat(dbPath); err != nil {
		return nil, GetStatisticsResult{}, fmt.Errorf("duckdb database for project %q does not exist. Call convert_db first", project)
	}

	dbConn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, GetStatisticsResult{}, fmt.Errorf("failed to open database: %w", err)
	}
	defer dbConn.Close()

	var totalPapers, totalAuthors, totalInstitutions, totalCountries int
	var missingCountries, missingInsts, imputedCountries int

	dbConn.QueryRow("SELECT COUNT(*) FROM papers").Scan(&totalPapers)
	dbConn.QueryRow("SELECT COUNT(*) FROM authors").Scan(&totalAuthors)
	dbConn.QueryRow("SELECT COUNT(*) FROM institutions").Scan(&totalInstitutions)
	dbConn.QueryRow("SELECT COUNT(DISTINCT country_code) FROM institutions").Scan(&totalCountries)

	dbConn.QueryRow("SELECT COUNT(*) FROM contributions WHERE country_code IS NULL OR country_code = ''").Scan(&missingCountries)
	dbConn.QueryRow("SELECT COUNT(*) FROM contributions WHERE institution_id IS NULL OR institution_id = ''").Scan(&missingInsts)
	dbConn.QueryRow("SELECT COUNT(*) FROM contributions WHERE country_code IS NOT NULL AND country_code != '' AND (imputed = true OR imputed = 1)").Scan(&imputedCountries)

	var completeness float64
	totalRows := 1
	dbConn.QueryRow("SELECT COUNT(*) FROM contributions").Scan(&totalRows)
	if totalRows > 0 {
		completeness = float64(totalRows-missingCountries) / float64(totalRows) * 100.0
	}

	return &mcp.CallToolResult{}, GetStatisticsResult{
		TotalPapers:            totalPapers,
		TotalAuthors:           totalAuthors,
		TotalInstitutions:      totalInstitutions,
		TotalCountries:         totalCountries,
		MissingCountryCount:    missingCountries,
		MissingInstitutionID:   missingInsts,
		ImputedCountryCount:    imputedCountries,
		ImputationCompleteness: completeness,
	}, nil
}

// WoS Integration handlers
func (s *APIServer) handleUploadWoSMCP(ctx context.Context, req *mcp.CallToolRequest, args UploadWoSArgs) (*mcp.CallToolResult, UploadWoSResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	if args.FilePath == "" {
		return nil, UploadWoSResult{}, fmt.Errorf("file_path is required")
	}

	ext := strings.ToLower(filepath.Ext(args.FilePath))
	if ext != ".csv" && ext != ".xlsx" && ext != ".xls" && ext != ".txt" {
		return nil, UploadWoSResult{}, fmt.Errorf("unsupported WoS format %q. Please upload a .csv, .xlsx, .xls, or .txt file", ext)
	}

	_, _, _, _, uploadDir := s.getProjectPaths(project)
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, UploadWoSResult{}, fmt.Errorf("failed to create uploads directory: %w", err)
	}

	srcFile, err := os.Open(args.FilePath)
	if err != nil {
		return nil, UploadWoSResult{}, fmt.Errorf("failed to open source WoS file: %w", err)
	}
	defer srcFile.Close()

	safeName := fmt.Sprintf("wos_%d%s", time.Now().UnixNano(), ext)
	destPath := filepath.Join(uploadDir, safeName)
	dst, err := os.Create(destPath)
	if err != nil {
		return nil, UploadWoSResult{}, fmt.Errorf("failed to create destination file in uploads: %w", err)
	}
	defer dst.Close()

	written, err := io.Copy(dst, srcFile)
	if err != nil {
		return nil, UploadWoSResult{}, fmt.Errorf("failed to write WoS file: %w", err)
	}

	return &mcp.CallToolResult{}, UploadWoSResult{
		Status:   "success",
		Filename: safeName,
		Size:     written,
	}, nil
}

func (s *APIServer) handleImportWoSDoisMCP(ctx context.Context, req *mcp.CallToolRequest, args ImportWoSDoisArgs) (*mcp.CallToolResult, ImportWoSDoisResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	if args.Filename == "" {
		return nil, ImportWoSDoisResult{}, fmt.Errorf("filename parameter is required")
	}

	_, _, _, dbDir, uploadsDir := s.getProjectPaths(project)
	filePath := filepath.Join(uploadsDir, args.Filename)
	dbPath := filepath.Join(dbDir, "papers.db")

	if _, err := os.Stat(filePath); err != nil {
		return nil, ImportWoSDoisResult{}, fmt.Errorf("file %q not found in uploads", args.Filename)
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	var err error
	if ext == ".csv" {
		err = wos.ImportWoSCSV(filePath, dbPath)
	} else if ext == ".xlsx" || ext == ".xls" {
		err = wos.ImportWoSExcel(filePath, dbPath)
	} else {
		return nil, ImportWoSDoisResult{}, fmt.Errorf("unsupported file extension %q for direct import", ext)
	}

	if err != nil {
		return nil, ImportWoSDoisResult{}, fmt.Errorf("WoS import failed: %w", err)
	}

	dbConn, err := sql.Open("duckdb", dbPath)
	var count int
	if err == nil {
		defer dbConn.Close()
		dbConn.QueryRow("SELECT COUNT(*) FROM wos_records").Scan(&count)
	}

	return &mcp.CallToolResult{}, ImportWoSDoisResult{
		Status:          "success",
		ImportedRecords: count,
	}, nil
}

func (s *APIServer) handleSyncWoSOpenAlexMCP(ctx context.Context, req *mcp.CallToolRequest, args SyncWoSOpenAlexArgs) (*mcp.CallToolResult, SyncWoSOpenAlexResult, error) {
	project := args.Project
	if project == "" {
		project = s.currentProject
	}

	if args.Filename == "" {
		return nil, SyncWoSOpenAlexResult{}, fmt.Errorf("filename is required")
	}

	_, _, _, dbDir, uploadsDir := s.getProjectPaths(project)
	filePath := filepath.Join(uploadsDir, args.Filename)
	dbPath := filepath.Join(dbDir, "papers.db")

	if _, err := os.Stat(filePath); err != nil {
		return nil, SyncWoSOpenAlexResult{}, fmt.Errorf("file %q not found in uploads", args.Filename)
	}

	report, err := wos.CompareDOIs(filePath, dbPath)
	if err != nil {
		return nil, SyncWoSOpenAlexResult{}, fmt.Errorf("sync comparison failed: %w", err)
	}

	return &mcp.CallToolResult{}, SyncWoSOpenAlexResult{
		TotalWoS:          report.TotalWoS,
		TotalDB:           report.TotalDB,
		ExactDOIMatches:   report.ExactDOIMatches,
		FuzzyTitleMatches: report.FuzzyTitleMatches,
		OverlapPercentage: report.OverlapPercent,
	}, nil
}
