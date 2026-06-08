package config

import (
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppConfig represents the root configuration structure mapped to collection.yml.
type AppConfig struct {
	API        APIConfig        `yaml:"api"`
	Filters    FiltersConfig    `yaml:"filters"`
	Keywords   string           `yaml:"keywords_file"`
	Topics     string           `yaml:"topics_file"`
	Anchors    string           `yaml:"anchor_file"`
	Collection CollectionConfig `yaml:"collection"`
	LLM        LLMConfig        `yaml:"llm"`
	Output     OutputConfig     `yaml:"output"`
}

// APIConfig defines api keys, endpoint URL, and contact email.
type APIConfig struct {
	Keys    []string `yaml:"keys"`
	GroqKey string   `yaml:"groq_key"`
	Email   string   `yaml:"email"`
	BaseURL string   `yaml:"base_url"`
}

// FiltersConfig defines date limits and document types.
type FiltersConfig struct {
	DateFrom string   `yaml:"date_from"`
	DateTo   string   `yaml:"date_to"`
	DocTypes []string `yaml:"doc_types"`
}

// CollectionConfig defines request tuning options.
type CollectionConfig struct {
	BatchSizeTopics    int `yaml:"batch_size_topics"`
	PerPage            int `yaml:"per_page"`
	ConcurrentRequests int `yaml:"concurrent_requests"`
	MaxRetries         int `yaml:"max_retries"`
	RetryDelay         int `yaml:"retry_delay"`
}

// LLMConfig defines configuration for local Ollama or Gemini models.
type LLMConfig struct {
	Provider string `yaml:"provider"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url"`
}

// OutputConfig defines output directory targets.
type OutputConfig struct {
	JSONLDir string `yaml:"jsonl_dir"`
	DBDir    string `yaml:"db_dir"`
}

// LoadConfig reads and parses collection.yml from the specified path.
func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig saves the configuration back to collection.yml.
func SaveConfig(path string, cfg *AppConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetKeywords reads the keywords string from the keywords.txt file, collapses whitespace, and returns it.
func GetKeywords(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`\s+`)
	cleaned := re.ReplaceAllString(string(data), " ")
	return strings.TrimSpace(cleaned), nil
}

// GetTopics parses the topics.txt file and returns a list of valid active topic IDs (ignoring lines starting with #).
func GetTopics(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var topics []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		topics = append(topics, line)
	}
	return topics, nil
}

// GetAnchors parses the anchor.txt file and returns DOIs or titles of papers that must be present in the collected data.
func GetAnchors(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var anchors []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		anchors = append(anchors, line)
	}
	return anchors, nil
}
