package wos

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"stratum/db"
	"github.com/xuri/excelize/v2"
)

func TestNormalizeDOI(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://doi.org/10.1103/physrevlett.120.010501", "10.1103/physrevlett.120.010501"},
		{"HTTP://DOI.ORG/10.1016/j.physletb.2019.01.002", "10.1016/j.physletb.2019.01.002"},
		{"doi:10.1016/j.physletb.2019.01.002", "10.1016/j.physletb.2019.01.002"},
		{"  10.1016/j.physletb.2019.01.002  ", "10.1016/j.physletb.2019.01.002"},
		{"", ""},
	}

	for _, tc := range tests {
		res := NormalizeDOI(tc.input)
		if res != tc.expected {
			t.Errorf("NormalizeDOI(%q): expected %q, got %q", tc.input, tc.expected, res)
		}
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hé (atoms) in strong: fields", "he atoms in strong fields"},
		{"Quantum   Supremacy  Demo!", "quantum supremacy demo"},
		{"Title with 123 numbers", "title with 123 numbers"},
		{"", ""},
	}

	for _, tc := range tests {
		res := NormalizeTitle(tc.input)
		if res != tc.expected {
			t.Errorf("NormalizeTitle(%q): expected %q, got %q", tc.input, tc.expected, res)
		}
	}
}

func TestImportWoSCSV(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_wos_csv_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.duckdb")
	mgr, err := db.NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("NewDBManager failed: %v", err)
	}

	if err := mgr.CreateSchema(); err != nil {
		mgr.Close()
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Insert existing paper
	_, err = mgr.RunQuery(`
		INSERT INTO papers (
			id, doi, title, publication_year, is_top_1_percent, is_top_10_percent
		) VALUES (
			'W42839485', '10.1103/physrevlett.120.010501', 'Quantum Supremacy Demo', 2018, FALSE, FALSE
		)
	`)
	if err != nil {
		mgr.Close()
		t.Fatalf("failed to insert test paper: %v", err)
	}
	mgr.Close()

	mockCSV := `Accession Number,DOI,Article Title,Authors,Source,Document Type,Publication Date,Times Cited,Top 1%,Top 10%
WOS:000123456700001,10.1103/physrevlett.120.010501,Quantum Supremacy Demo,"Smith, A.",Physical Review Letters,Article,2018,42,1,1
WOS:000123456700002,10.1016/j.physletb.2019.01.002,Another Quantum Sensational Paper,"Doe, J.",Physics Letters B,Article,2019,10,0,1
`
	csvPath := filepath.Join(tmpDir, "US.csv")
	if err := os.WriteFile(csvPath, []byte(mockCSV), 0644); err != nil {
		t.Fatalf("failed to write CSV: %v", err)
	}

	// Import CSV
	if err := ImportWoSCSV(csvPath, dbPath); err != nil {
		t.Fatalf("ImportWoSCSV failed: %v", err)
	}

	// Verify updates
	dbConn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db connection: %v", err)
	}
	defer dbConn.Close()

	// Check that existing paper was updated
	var fromWoS bool
	var top1, top10 bool
	var wosAcc sql.NullString
	err = dbConn.QueryRow("SELECT from_wos, is_top_1_percent, is_top_10_percent, wos_accession_number FROM papers WHERE id = 'W42839485'").Scan(&fromWoS, &top1, &top10, &wosAcc)
	if err != nil {
		t.Fatalf("failed to query updated paper: %v", err)
	}
	if !fromWoS {
		t.Error("expected from_wos to be true")
	}
	if !top1 || !top10 {
		t.Errorf("expected top percentiles to be OR-upgraded to true, got top1=%v, top10=%v", top1, top10)
	}
	if wosAcc.String != "WOS:000123456700001" {
		t.Errorf("expected wos_accession_number 'WOS:000123456700001', got %q", wosAcc.String)
	}

	// Check new stub paper was inserted
	var newTitle string
	err = dbConn.QueryRow("SELECT title FROM papers WHERE id = 'WOS_000123456700002'").Scan(&newTitle)
	if err != nil {
		t.Fatalf("failed to query new paper: %v", err)
	}
	if newTitle != "Another Quantum Sensational Paper" {
		t.Errorf("expected title 'Another Quantum Sensational Paper', got %q", newTitle)
	}

	// Check contribution country marker
	var count int
	dbConn.QueryRow("SELECT COUNT(*) FROM contributions WHERE source = 'wos' AND country_code = 'US'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 WoS country contributions, got %d", count)
	}
}

func createMockExcel(t *testing.T, path string) {
	f := excelize.NewFile()
	defer f.Close()

	headers := []string{"Accession Number", "DOI", "Article Title", "Authors", "Source", "Document Type", "Publication Date", "Times Cited", "Top 1%", "Top 10%"}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue("Sheet1", cell, h)
	}

	row2 := []interface{}{"WOS:000123456700001", "10.1103/physrevlett.120.010501", "Quantum Supremacy Demo", "Smith, A.", "Physical Review Letters", "Article", 2018, 42, 1, 1}
	for col, v := range row2 {
		cell, _ := excelize.CoordinatesToCellName(col+1, 2)
		f.SetCellValue("Sheet1", cell, v)
	}

	row3 := []interface{}{"WOS:000123456700003", "", "Fuzzy Quantum Metrology Title That Is At Least Thirty Characters Long", "Doe, A.", "Nature Physics", "Article", 2021, 5, 0, 0}
	for col, v := range row3 {
		cell, _ := excelize.CoordinatesToCellName(col+1, 3)
		f.SetCellValue("Sheet1", cell, v)
	}

	if err := f.SaveAs(path); err != nil {
		t.Fatalf("failed to save mock Excel: %v", err)
	}
}

func TestImportWoSExcel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_wos_excel_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.duckdb")
	mgr, err := db.NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("NewDBManager failed: %v", err)
	}

	if err := mgr.CreateSchema(); err != nil {
		mgr.Close()
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Insert existing paper for fuzzy title match testing
	_, err = mgr.RunQuery(`
		INSERT INTO papers (
			id, doi, title, publication_year, is_top_1_percent, is_top_10_percent
		) VALUES (
			'W_FUZZY', NULL, 'Fuzzy Quantum Metrology Title That Is At Least Thirty Characters Long', 2021, FALSE, FALSE
		)
	`)
	if err != nil {
		mgr.Close()
		t.Fatalf("failed to insert fuzzy test paper: %v", err)
	}
	mgr.Close()

	excelPath := filepath.Join(tmpDir, "Canada.xlsx")
	createMockExcel(t, excelPath)

	if err := ImportWoSExcel(excelPath, dbPath); err != nil {
		t.Fatalf("ImportWoSExcel failed: %v", err)
	}

	dbConn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		t.Fatalf("failed to open db connection: %v", err)
	}
	defer dbConn.Close()

	// Verify that fuzzy matched paper got updated
	var fromWoS bool
	var wosAcc sql.NullString
	err = dbConn.QueryRow("SELECT from_wos, wos_accession_number FROM papers WHERE id = 'W_FUZZY'").Scan(&fromWoS, &wosAcc)
	if err != nil {
		t.Fatalf("failed to query fuzzy matched paper: %v", err)
	}
	if !fromWoS {
		t.Error("expected fuzzy matched paper from_wos to be true")
	}
	if wosAcc.String != "WOS:000123456700003" {
		t.Errorf("expected accession 'WOS:000123456700003', got %q", wosAcc.String)
	}

	// Verify contribution country code (Canada -> CA)
	var count int
	dbConn.QueryRow("SELECT COUNT(*) FROM contributions WHERE source = 'wos' AND country_code = 'CA'").Scan(&count)
	if count != 2 {
		t.Errorf("expected 2 WoS contributions for CA, got %d", count)
	}
}

func TestCompareDOIs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_wos_compare_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.duckdb")
	mgr, err := db.NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("NewDBManager failed: %v", err)
	}

	if err := mgr.CreateSchema(); err != nil {
		mgr.Close()
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Insert existing papers
	_, err = mgr.RunQuery(`
		INSERT INTO papers (id, doi, title, publication_year) VALUES
		('W1', '10.1103/physrevlett.120.010501', 'Quantum Supremacy Demo', 2018),
		('W2', NULL, 'Fuzzy Quantum Metrology Title That Is At Least Thirty Characters Long', 2021)
	`)
	if err != nil {
		mgr.Close()
		t.Fatalf("failed to insert test papers: %v", err)
	}
	mgr.Close()

	mockCSV := `Accession Number,DOI,Article Title,Authors,Source,Document Type,Publication Date,Times Cited,Top 1%,Top 10%
WOS:0001,10.1103/physrevlett.120.010501,Quantum Supremacy Demo,Smith, A.,Physical Review Letters,Article,2018,42,1,1
WOS:0002,,Fuzzy Quantum Metrology Title That Is At Least Thirty Characters Long,Doe, J.,Physics Letters B,Article,2021,10,0,1
WOS:0003,10.9999/not-existing-doi,Non Existing Title,Doe, J.,Physics Letters B,Article,2022,10,0,1
`
	csvPath := filepath.Join(tmpDir, "compare.csv")
	if err := os.WriteFile(csvPath, []byte(mockCSV), 0644); err != nil {
		t.Fatalf("failed to write CSV: %v", err)
	}

	report, err := CompareDOIs(csvPath, dbPath)
	if err != nil {
		t.Fatalf("CompareDOIs failed: %v", err)
	}

	if report.TotalWoS != 3 {
		t.Errorf("expected TotalWoS = 3, got %d", report.TotalWoS)
	}
	if report.TotalDB != 2 {
		t.Errorf("expected TotalDB = 2, got %d", report.TotalDB)
	}
	if report.ExactDOIMatches != 1 {
		t.Errorf("expected ExactDOIMatches = 1, got %d", report.ExactDOIMatches)
	}
	if report.FuzzyTitleMatches != 1 {
		t.Errorf("expected FuzzyTitleMatches = 1, got %d", report.FuzzyTitleMatches)
	}
	diff := report.OverlapPercent - (2.0 / 3.0 * 100.0)
	if diff < -0.0001 || diff > 0.0001 {
		t.Errorf("expected OverlapPercent = 66.666, got %f", report.OverlapPercent)
	}
}
