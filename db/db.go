package db

import (
	"database/sql"
	"fmt"

	_ "github.com/duckdb/duckdb-go/v2"
)

// DBManager wraps the DuckDB connection and handles schema initialization and batch insertions.
type DBManager struct {
	dbPath string
	db     *sql.DB
}

// LoadStats records loaded rows and error counts during the JSONL processing.
type LoadStats struct {
	Papers        int `json:"papers"`
	Authors       int `json:"authors"`
	Institutions  int `json:"institutions"`
	Countries     int `json:"countries"`
	Contributions int `json:"contributions"`
	Errors        int `json:"errors"`
	Skipped       int `json:"skipped"`
}

// DashboardStats stores top-level aggregate values to power the web dashboard.
type DashboardStats struct {
	TotalPapers       int            `json:"total_papers"`
	TotalAuthors      int            `json:"total_authors"`
	TotalInstitutions int            `json:"total_institutions"`
	TotalCountries    int            `json:"total_countries"`
	PapersByYear      []YearStat     `json:"papers_by_year"`
	OAStatusCounts    []OAStatusStat `json:"oa_status_counts"`
	TopJournals       []JournalStat  `json:"top_journals"`
	CountryCounts     []CountryStat  `json:"country_counts"`
}

type YearStat struct {
	Year  int `json:"year"`
	Count int `json:"count"`
}

type OAStatusStat struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type JournalStat struct {
	JournalName string `json:"journal_name"`
	Count       int    `json:"count"`
}

type CountryStat struct {
	CountryCode string `json:"country_code"`
	Count       int    `json:"count"`
}

// NewDBManager initializes a new database manager.
func NewDBManager(dbPath string) (*DBManager, error) {
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return &DBManager{
		dbPath: dbPath,
		db:     db,
	}, nil
}

// Close closes the underlying DuckDB database connection.
func (m *DBManager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

// CreateSchema initializes the database tables (papers, authors, institutions, countries, contributions).
func (m *DBManager) CreateSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS papers (
			id VARCHAR PRIMARY KEY, doi VARCHAR, title TEXT,
			publication_year INTEGER, publication_date VARCHAR, type VARCHAR,
			journal_name VARCHAR, journal_issn VARCHAR, is_core_journal BOOLEAN,
			publisher VARCHAR, is_oa BOOLEAN, oa_status VARCHAR, oa_url VARCHAR,
			cited_by_count INTEGER, citation_percentile DOUBLE,
			is_top_1_percent BOOLEAN, is_top_10_percent BOOLEAN, fwci DOUBLE,
			primary_topic_id VARCHAR, primary_topic_name VARCHAR, primary_topic_score DOUBLE,
			primary_topic_field VARCHAR, primary_topic_subfield VARCHAR, primary_topic_domain VARCHAR,
			institutions_distinct_count INTEGER, countries_distinct_count INTEGER,
			is_international BOOLEAN, abstract_text TEXT, updated_date VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS authors (
			id VARCHAR PRIMARY KEY, display_name VARCHAR, orcid VARCHAR
		)`,
		`CREATE TABLE IF NOT EXISTS institutions (
			id VARCHAR PRIMARY KEY, display_name VARCHAR,
			country_code VARCHAR, type VARCHAR, ror_id VARCHAR,
			is_synthetic BOOLEAN DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS countries (
			id INTEGER PRIMARY KEY, country_name VARCHAR, country_code VARCHAR UNIQUE, status INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS contributions (
			row_id INTEGER PRIMARY KEY, paper_id VARCHAR, author_id VARCHAR,
			institution_id VARCHAR, country_code VARCHAR, author_name VARCHAR,
			author_position VARCHAR, is_corresponding BOOLEAN, raw_affiliation_string VARCHAR
		)`,
	}

	for _, q := range queries {
		if _, err := m.db.Exec(q); err != nil {
			return err
		}
	}

	// Initialize contribution sequence seq_contrib
	var maxID int
	err := m.db.QueryRow("SELECT COALESCE(MAX(row_id), 0) FROM contributions").Scan(&maxID)
	if err != nil {
		maxID = 0
	}
	m.db.Exec("DROP SEQUENCE IF EXISTS seq_contrib")
	_, err = m.db.Exec(fmt.Sprintf("CREATE SEQUENCE seq_contrib START %d", maxID+1))
	return err
}

// LoadJSONL parses a downloaded JSONL file and loads normalized records into DuckDB with progress updates.
func (m *DBManager) LoadJSONL(jsonlPath string, progressChan chan<- int) (*LoadStats, error) {
	// TODO: Scan JSONL, normalize records, execute bulk transactions.
	return nil, nil
}

// GetDashboardStats queries aggregates to get total counts and analytical breakdown statistics.
func (m *DBManager) GetDashboardStats() (*DashboardStats, error) {
	// TODO: Run SQL queries to compile metrics.
	return nil, nil
}

// RunQuery executes a custom SQL statement on the DuckDB database and returns a generic array of row maps.
func (m *DBManager) RunQuery(query string) ([]map[string]interface{}, error) {
	// TODO: Execute query, read columns and types dynamically, and return rows.
	return nil, nil
}
