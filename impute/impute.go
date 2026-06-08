package impute

import (
	"context"
)

// ImputationEngine coordinates imputation pipelines over missing institutions and countries.
type ImputationEngine struct {
	dbPath string
}

// NewImputationEngine initializes a new engine.
func NewImputationEngine(dbPath string) *ImputationEngine {
	return &ImputationEngine{dbPath: dbPath}
}

// NormalizeCountryCode standardizes country codes, e.g. mapping "HK" to "CN".
func NormalizeCountryCode(code string) string {
	// TODO: Clean whitespace, capitalize, and map overrides like HK->CN.
	return ""
}

// CountryInference holds the result of mapping a raw affiliation string to a country.
type CountryInference struct {
	CountryCode  string
	Status       string // unambiguous, ambiguous, none
	MatchedTerms []string
}

// InferCountryFromAffiliation uses country name aliases and regex word-boundary checks to identify a country.
func InferCountryFromAffiliation(rawAffiliation string) *CountryInference {
	// TODO: Match raw text against country names list and return inference.
	return nil
}

// SyntheticInstitutionID generates a stable hex ID prefixed with IMP_ from a display name.
func SyntheticInstitutionID(name string) string {
	// TODO: Collapse spaces, SHA1 hash name, take first 10 hex chars, and prefix IMP_.
	return ""
}

// InstitutionRecord represents an existing database institution entry for indexing.
type InstitutionRecord struct {
	ID          string
	DisplayName string
	CountryCode string
}

// InstitutionMatch represents the result of a similarity match query.
type InstitutionMatch struct {
	InstitutionID string
	DisplayName   string
	CountryCode   string
	Score         float64
}

// InstitutionMatcher wraps sentence-transformer embeddings to run similarity lookups.
type InstitutionMatcher struct {
	ModelName string
	Records   []InstitutionRecord
}

// NewInstitutionMatcher initializes the matcher.
func NewInstitutionMatcher(modelName string) *InstitutionMatcher {
	return &InstitutionMatcher{ModelName: modelName}
}

// Index generates or loads cached vector embeddings for the provided database records.
func (m *InstitutionMatcher) Index(records []InstitutionRecord) error {
	// TODO: Lazy-load embeddings models and index names.
	return nil
}

// FindMatch returns the best matched institution if above the threshold.
func (m *InstitutionMatcher) FindMatch(query string, threshold float64) *InstitutionMatch {
	// TODO: Perform similarity match and return if above threshold.
	return nil
}

// TopK returns the top K most similar institutions for a query.
func (m *InstitutionMatcher) TopK(query string, k int) ([]InstitutionMatch, error) {
	// TODO: Run similarity comparison and return slice.
	return nil, nil
}

// ImputeCrossRef fetches missing raw_affiliation_string entries from Crossref using paper DOIs.
func (e *ImputationEngine) ImputeCrossRef(ctx context.Context, progressChan chan<- int) error {
	// TODO: Fetch eligible DOIs, request CrossRef API, parse response, and update contributions table.
	return nil
}

// ImputeLLM uses an LLM (Ollama or Gemini) to extract institution and country names from raw affiliation strings.
func (e *ImputationEngine) ImputeLLM(ctx context.Context, provider string, model string, baseURL string, progressChan chan<- int) error {
	// TODO: Pull missing entries, batch-request LLM completions, run similarity string matches, and save in DB.
	return nil
}

// ImputePDF downloads the paper PDF (e.g. via arXiv/Unpaywall), extracts text, and extracts affiliation using LLM.
func (e *ImputationEngine) ImputePDF(ctx context.Context, provider string, model string, baseURL string, limit int, progressChan chan<- int) error {
	// TODO: Fetch eligible papers, resolve PDF URL, download PDF, extract first-page text, prompt LLM, and update DB.
	return nil
}

// OllamaClient triggers local Ollama completions.
type OllamaClient struct {
	BaseURL string
	Model   string
}

// Complete submits a prompt payload to the Ollama endpoint.
func (c *OllamaClient) Complete(ctx context.Context, prompt string) (string, error) {
	// TODO: POST request to /api/generate or /api/chat.
	return "", nil
}

// GeminiClient wraps the google.golang.org/genai client.
type GeminiClient struct {
	APIKey string
	Model  string
}

// Complete submits a prompt payload to the Gemini API.
func (c *GeminiClient) Complete(ctx context.Context, prompt string) (string, error) {
	// TODO: Call Google GenAI SDK to generate content.
	return "", nil
}
