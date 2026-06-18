package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseXLSRows(t *testing.T) {
	// Locate any .xls file in the uploads directory relative to api/
	uploadsDir := filepath.Join("..", "data", "uploads")
	files, err := os.ReadDir(uploadsDir)
	if err != nil {
		t.Skip("Skipping TestParseXLSRows: uploads directory not found relative to api package:", err)
	}

	var xlsFile string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".xls") {
			xlsFile = filepath.Join(uploadsDir, f.Name())
			break
		}
	}

	if xlsFile == "" {
		t.Skip("Skipping TestParseXLSRows: no .xls files found in uploads directory")
	}

	t.Logf("Testing parseXLSRows with file: %s", xlsFile)
	rows, err := parseXLSRows(xlsFile)
	if err != nil {
		t.Fatalf("parseXLSRows failed: %v", err)
	}

	if len(rows) == 0 {
		t.Fatal("Expected rows to be parsed, but got 0 rows")
	}

	headers := rows[0]
	t.Logf("Parsed headers: %v", headers)
	if len(headers) == 0 {
		t.Fatal("Expected at least one column/header in XLS file")
	}

	// Verify parseExcelHeaders
	h, err := parseExcelHeaders(xlsFile)
	if err != nil {
		t.Fatalf("parseExcelHeaders failed: %v", err)
	}
	if len(h) != len(headers) {
		t.Errorf("parseExcelHeaders returned %d columns, expected %d", len(h), len(headers))
	}
	for i, col := range h {
		if col != headers[i] {
			t.Errorf("Mismatch at column %d: got %q, expected %q", i, col, headers[i])
		}
	}
}

func TestExtractDOIsFromExcelRobust(t *testing.T) {
	uploadsDir := filepath.Join("..", "data", "uploads")
	files, err := os.ReadDir(uploadsDir)
	if err != nil {
		t.Skip("Skipping TestExtractDOIsFromExcelRobust: uploads directory not found:", err)
	}

	var xlsFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".xls") {
			xlsFiles = append(xlsFiles, filepath.Join(uploadsDir, f.Name()))
		}
	}

	if len(xlsFiles) == 0 {
		t.Skip("Skipping TestExtractDOIsFromExcelRobust: no .xls files found")
	}

	for _, xlsFile := range xlsFiles {
		t.Run(filepath.Base(xlsFile), func(t *testing.T) {
			t.Logf("Testing extractDOIsFromExcel with file: %s", xlsFile)
			headers, err := parseExcelHeaders(xlsFile)
			if err != nil {
				t.Fatalf("Failed to parse headers: %v", err)
			}

			var doiCol string
			for _, h := range headers {
				if strings.EqualFold(h, "doi") {
					doiCol = h
					break
				}
			}
			if doiCol == "" {
				for _, h := range headers {
					if strings.Contains(strings.ToLower(h), "doi") {
						doiCol = h
						break
					}
				}
			}
			if doiCol == "" {
				doiCol = "doi"
			}

			t.Logf("Found DOI column name: %q", doiCol)

			dois, err := extractDOIsFromExcel(xlsFile, doiCol)
			if err != nil {
				t.Fatalf("extractDOIsFromExcel failed: %v", err)
			}

			t.Logf("Extracted %d DOIs from file", len(dois))

			// The user mentions only 22 DOIs were loaded initially, but there should be much more.
			// Let's assert that we loaded significantly more than 22 DOIs.
			if len(dois) <= 22 {
				t.Errorf("Expected to extract more than 22 DOIs, but only extracted %d. The row-scanning fallback might not be working.", len(dois))
			}
		})
	}
}

