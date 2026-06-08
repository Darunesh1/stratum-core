package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDBManagerAndClose(t *testing.T) {
	// Create temporary directory for test database
	tmpDir, err := os.MkdirTemp("", "stratum_db_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.duckdb")

	// Verify manager creation
	mgr, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("NewDBManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("expected DBManager instance, got nil")
	}

	if mgr.dbPath != dbPath {
		t.Errorf("expected dbPath '%s', got '%s'", dbPath, mgr.dbPath)
	}

	if mgr.db == nil {
		t.Error("expected database connection, got nil")
	}

	// Verify manager close
	if err := mgr.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify database file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Errorf("expected database file to exist at %s, but it does not", dbPath)
	}
}

func TestCreateSchema(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_db_schema_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.duckdb")

	mgr, err := NewDBManager(dbPath)
	if err != nil {
		t.Fatalf("NewDBManager failed: %v", err)
	}
	defer mgr.Close()

	if err := mgr.CreateSchema(); err != nil {
		t.Fatalf("CreateSchema failed: %v", err)
	}

	// Verify tables are created in main schema
	expectedTables := map[string]bool{
		"papers":        false,
		"authors":       false,
		"institutions":  false,
		"countries":     false,
		"contributions": false,
	}

	rows, err := mgr.db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'")
	if err != nil {
		t.Fatalf("failed to query information_schema: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		if _, ok := expectedTables[name]; ok {
			expectedTables[name] = true
		}
	}

	for k, found := range expectedTables {
		if !found {
			t.Errorf("expected table '%s' was not found in schema", k)
		}
	}
}
