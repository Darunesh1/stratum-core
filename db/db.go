package db

import (
	"database/sql"
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
	// TODO: Connect to duckdb using the official driver.
	return nil, nil
}

// Close closes the underlying DuckDB database connection.
func (m *DBManager) Close() error {
	// TODO: Close database.
	return nil
}

// CreateSchema initializes the database tables (papers, authors, institutions, countries, contributions).
func (m *DBManager) CreateSchema() error {
	// TODO: Execute CREATE TABLE DDL queries.
	return nil
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
