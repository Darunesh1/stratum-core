package workflow

import (
	"database/sql"
	"stratum/config"
)

type ProjectSummary struct {
	Project    string `json:"project"`
	Status     string `json:"status"`
	Papers     int    `json:"papers"`
	Downloaded bool   `json:"downloaded"`
	LastAction string `json:"last_action"`
	NextAction string `json:"next_action"`
}

type WorkflowState struct {
	Stage     string   `json:"stage"`
	Completed []string `json:"completed"`
	Pending   []string `json:"pending"`
	Running   bool     `json:"running"`
}

func AnalyzeState(cfg *config.AppConfig, dbConn *sql.DB) (*ProjectSummary, *WorkflowState, string, error) {
	var paperCount int
	var hasTable bool

	if dbConn != nil {
		err := dbConn.QueryRow("SELECT COUNT(*) FROM papers").Scan(&paperCount)
		if err == nil {
			hasTable = true
		}
	}

	stage := "Initialization"
	var completed []string
	var pending []string

	completed = append(completed, "Initialization")

	if cfg.Keywords != "" {
		completed = append(completed, "Keywords Configured")
	} else {
		pending = append(pending, "Configure Keywords")
	}

	if len(cfg.Anchors) > 0 {
		completed = append(completed, "Anchors Seeded")
	} else {
		pending = append(pending, "Add Anchor DOIs")
	}

	if len(cfg.Topics) > 0 {
		completed = append(completed, "Topics Configured")
	} else {
		pending = append(pending, "Filter Topics")
	}

	downloaded := false
	if hasTable && paperCount > 0 {
		downloaded = true
		completed = append(completed, "Ingested Papers")
	} else {
		pending = append(pending, "Download & Ingest Papers")
	}

	imputed := false
	var missingCount int
	if hasTable && downloaded {
		err := dbConn.QueryRow("SELECT COUNT(*) FROM contributions WHERE institution_id IS NULL AND country_code IS NULL").Scan(&missingCount)
		if err == nil && missingCount == 0 {
			imputed = true
			completed = append(completed, "Enriched Metadata")
		} else {
			pending = append(pending, "Run Imputation")
		}
	}

	var nextAction string
	var status string
	var lastAction string

	if cfg.Keywords == "" {
		stage = "Query Setup"
		nextAction = "Define boolean search keywords (using upload_file/extract_query or configure_project)."
		status = "Unconfigured"
		lastAction = "Project Created"
	} else if len(cfg.Anchors) == 0 {
		stage = "Validation Setup"
		nextAction = "Add must-find anchor DOIs to test query recall."
		status = "Keywords Configured"
		lastAction = "Keywords Defined"
	} else if len(cfg.Topics) == 0 {
		stage = "Topic Refinement"
		nextAction = "Run search/get_topics to discover fields, and add primary topic ID filters."
		status = "Anchors Seeded"
		lastAction = "Anchors Added"
	} else if !downloaded {
		stage = "Search Validation"
		nextAction = "Review a random sample using get_sample, then call download followed by convert_db to ingest."
		status = "Search Filters Set"
		lastAction = "Filters Configured"
	} else if !imputed && missingCount > 0 {
		stage = "Metadata Imputation"
		nextAction = "Run metadata imputation (crossref, llm, pdf) to resolve missing country and institution IDs."
		status = "Ingestion Complete"
		lastAction = "Database Loaded"
	} else {
		stage = "Analysis & Queries"
		nextAction = "Dataset is fully ready. Execute TF-IDF term extractions or run custom SQL reports."
		status = "Enrichment Complete"
		lastAction = "Imputation Finished"
	}

	summary := &ProjectSummary{
		Status:     status,
		Papers:     paperCount,
		Downloaded: downloaded,
		LastAction: lastAction,
		NextAction: nextAction,
	}

	state := &WorkflowState{
		Stage:     stage,
		Completed: completed,
		Pending:   pending,
		Running:   false,
	}

	return summary, state, nextAction, nil
}
