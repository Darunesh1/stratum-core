package wos

// ComparisonReport provides metadata statistics on overlap between DuckDB papers and WoS papers.
type ComparisonReport struct {
	TotalWoS          int     `json:"total_wos"`
	TotalDB           int     `json:"total_db"`
	ExactDOIMatches   int     `json:"exact_doi_matches"`
	FuzzyTitleMatches int     `json:"fuzzy_title_matches"`
	OverlapPercent    float64 `json:"overlap_percent"`
}

// NormalizeDOI strips URL prefixes, lowercases, and cleans DOI strings.
func NormalizeDOI(raw string) string {
	// TODO: Strip https://doi.org/, http://doi.org/, doi: prefixes and return lowercase trimmed.
	return ""
}

// NormalizeTitle performs aggressive normalization: strips diacritics, punctuation, lowercases, and collapses spaces.
func NormalizeTitle(raw string) string {
	// TODO: Normalize NFKD, remove diacritics, lowercase, remove punctuation, collapse spaces.
	return ""
}

// ImportWoSCSV loads Web of Science CSV datasets, mapping and loading relevant records.
func ImportWoSCSV(csvPath string, dbPath string) error {
	// TODO: Load CSV, parse, and insert details into the database.
	return nil
}

// ImportWoSExcel reads Web of Science Excel exports (.xlsx) and streams records into the database.
func ImportWoSExcel(excelPath string, dbPath string) error {
	// TODO: Open xlsx sheet, map columns, and load into DuckDB.
	return nil
}

// CompareDOIs queries the DuckDB database to check which DOIs (and fuzzy matched titles) from the WoS file overlap.
func CompareDOIs(wosCSVPath string, dbPath string) (*ComparisonReport, error) {
	// TODO: Compare DOIs and titles, calculate overlap percentage, and return report.
	return nil, nil
}
