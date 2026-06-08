package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "stratum_config_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	yamlContent := `
api:
  keys:
    - "key1"
    - "key2"
  groq_key: "gsk_test"
  email: "test@domain.com"
  base_url: "https://api.openalex.org"
filters:
  date_from: "2020-01-01"
  date_to: "2024-01-01"
  doc_types:
    - "article"
keywords_file: "config/keywords.txt"
topics_file: "config/topics.txt"
anchor_file: "config/anchor.txt"
collection:
  batch_size_topics: 5
  per_page: 100
  concurrent_requests: 2
  max_retries: 3
  retry_delay: 1
llm:
  provider: "ollama"
  model: "qwen-test"
  base_url: "http://localhost:11434"
output:
  jsonl_dir: "data/raw/"
  db_dir: "data/db/"
`
	configPath := filepath.Join(tmpDir, "collection.yml")
	if err := os.WriteFile(configPath, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.API.Email != "test@domain.com" {
		t.Errorf("expected email 'test@domain.com', got '%s'", cfg.API.Email)
	}
	if len(cfg.API.Keys) != 2 || cfg.API.Keys[0] != "key1" || cfg.API.Keys[1] != "key2" {
		t.Errorf("unexpected API keys: %v", cfg.API.Keys)
	}
	if cfg.Filters.DateFrom != "2020-01-01" {
		t.Errorf("expected DateFrom '2020-01-01', got '%s'", cfg.Filters.DateFrom)
	}
	if cfg.Collection.PerPage != 100 {
		t.Errorf("expected PerPage 100, got %d", cfg.Collection.PerPage)
	}
	if cfg.LLM.Model != "qwen-test" {
		t.Errorf("expected LLM model 'qwen-test', got '%s'", cfg.LLM.Model)
	}
	if cfg.Output.DBDir != "data/db/" {
		t.Errorf("expected DBDir 'data/db/', got '%s'", cfg.Output.DBDir)
	}

	// Test loading non-existent file returns error
	_, err = LoadConfig(filepath.Join(tmpDir, "doesnotexist.yml"))
	if err == nil {
		t.Error("expected error loading non-existent file, got nil")
	}
}

func TestSaveConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_config_save_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := &AppConfig{
		API: APIConfig{
			Keys:    []string{"savekey"},
			GroqKey: "groqkey",
			Email:   "save@test.com",
			BaseURL: "https://base.org",
		},
		Filters: FiltersConfig{
			DateFrom: "2019-12-31",
			DateTo:   "2023-12-31",
			DocTypes: []string{"proceedings"},
		},
		Keywords: "kw.txt",
		Topics:   "tp.txt",
		Anchors:  "ac.txt",
		Collection: CollectionConfig{
			BatchSizeTopics:    10,
			PerPage:            200,
			ConcurrentRequests: 5,
			MaxRetries:         4,
			RetryDelay:         2,
		},
		LLM: LLMConfig{
			Provider: "groq",
			Model:    "llama-save",
			BaseURL:  "https://api.groq.com",
		},
		Output: OutputConfig{
			JSONLDir: "raw/",
			DBDir:    "db/",
		},
	}

	configPath := filepath.Join(tmpDir, "saved.yml")
	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load back and assert
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	if loaded.API.Email != "save@test.com" {
		t.Errorf("expected email 'save@test.com', got '%s'", loaded.API.Email)
	}
	if len(loaded.API.Keys) != 1 || loaded.API.Keys[0] != "savekey" {
		t.Errorf("unexpected keys: %v", loaded.API.Keys)
	}
	if loaded.Filters.DateFrom != "2019-12-31" {
		t.Errorf("expected date_from '2019-12-31', got '%s'", loaded.Filters.DateFrom)
	}
	if loaded.Collection.PerPage != 200 {
		t.Errorf("expected PerPage 200, got %d", loaded.Collection.PerPage)
	}
	if loaded.LLM.Model != "llama-save" {
		t.Errorf("expected LLM model 'llama-save', got '%s'", loaded.LLM.Model)
	}
}

func TestGetKeywords(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_config_keywords_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	rawKeywords := `
(
  ("quantum computing" OR "quantum computation")
  AND
  ("qubit" OR "quantum gate")
)
`
	filePath := filepath.Join(tmpDir, "keywords.txt")
	if err := os.WriteFile(filePath, []byte(rawKeywords), 0644); err != nil {
		t.Fatalf("failed to write keywords file: %v", err)
	}

	cleaned, err := GetKeywords(filePath)
	if err != nil {
		t.Fatalf("GetKeywords failed: %v", err)
	}

	expected := `( ("quantum computing" OR "quantum computation") AND ("qubit" OR "quantum gate") )`
	if cleaned != expected {
		t.Errorf("expected '%s', got '%s'", expected, cleaned)
	}

	// Test non-existent file
	_, err = GetKeywords(filepath.Join(tmpDir, "nonexistent_keywords.txt"))
	if err == nil {
		t.Error("expected error for non-existent keywords file, got nil")
	}
}

func TestGetTopics(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_config_topics_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	topicsContent := `
# Active topics for Quantum Sensing and Metrology
T10020
T10682
  # Indented comment
  T12345
`
	filePath := filepath.Join(tmpDir, "topics.txt")
	if err := os.WriteFile(filePath, []byte(topicsContent), 0644); err != nil {
		t.Fatalf("failed to write topics file: %v", err)
	}

	topics, err := GetTopics(filePath)
	if err != nil {
		t.Fatalf("GetTopics failed: %v", err)
	}

	if len(topics) != 3 {
		t.Errorf("expected 3 topics, got %d", len(topics))
	}
	expected := []string{"T10020", "T10682", "T12345"}
	for i, v := range topics {
		if v != expected[i] {
			t.Errorf("expected topics[%d] to be '%s', got '%s'", i, expected[i], v)
		}
	}

	// Test non-existent file returns error
	_, err = GetTopics(filepath.Join(tmpDir, "nonexistent_topics.txt"))
	if err == nil {
		t.Error("expected error for non-existent topics file, got nil")
	}
}

func TestGetAnchors(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stratum_config_anchors_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	anchorsContent := `
# Anchor papers DOI/titles
10.1038/nphys1170
Quantum supremacy using a programmable superconducting processor
`
	filePath := filepath.Join(tmpDir, "anchor.txt")
	if err := os.WriteFile(filePath, []byte(anchorsContent), 0644); err != nil {
		t.Fatalf("failed to write anchors file: %v", err)
	}

	anchors, err := GetAnchors(filePath)
	if err != nil {
		t.Fatalf("GetAnchors failed: %v", err)
	}

	if len(anchors) != 2 {
		t.Errorf("expected 2 anchors, got %d", len(anchors))
	}
	expected := []string{"10.1038/nphys1170", "Quantum supremacy using a programmable superconducting processor"}
	for i, v := range anchors {
		if v != expected[i] {
			t.Errorf("expected anchors[%d] to be '%s', got '%s'", i, expected[i], v)
		}
	}

	// Test non-existent file returns error
	_, err = GetAnchors(filepath.Join(tmpDir, "nonexistent_anchors.txt"))
	if err == nil {
		t.Error("expected error for non-existent anchors file, got nil")
	}
}
