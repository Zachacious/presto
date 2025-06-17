package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration
type Config struct {
	AI       AIConfig      `yaml:"ai"`
	Defaults DefaultConfig `yaml:"defaults"`
	Filters  FilterConfig  `yaml:"filters"`
}

type AIConfig struct {
	Provider    string        `yaml:"provider"`
	Model       string        `yaml:"model"`
	APIKeyEnv   string        `yaml:"api_key_env"`
	BaseURL     string        `yaml:"base_url"`
	MaxTokens   int           `yaml:"max_tokens"`
	Temperature float64       `yaml:"temperature"`
	Timeout     time.Duration `yaml:"timeout"`
}

type DefaultConfig struct {
	MaxConcurrent  int    `yaml:"max_concurrent"`
	OutputMode     string `yaml:"output_mode"`
	OutputSuffix   string `yaml:"output_suffix"`
	BackupOriginal bool   `yaml:"backup_original"`
	RemoveComments bool   `yaml:"remove_comments"`
	FilePattern    string `yaml:"file_pattern"`
}

type FilterConfig struct {
	MaxFileSize  int64    `yaml:"max_file_size"` // in bytes
	ExcludeDirs  []string `yaml:"exclude_dirs"`
	ExcludeFiles []string `yaml:"exclude_files"`
	IncludeExts  []string `yaml:"include_exts"`
	ExcludeExts  []string `yaml:"exclude_exts"`
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	return &Config{
		AI: AIConfig{
			Provider:    "openrouter",
			Model:       "anthropic/claude-3.5-sonnet",
			APIKeyEnv:   "OPENROUTER_API_KEY",
			BaseURL:     "https://openrouter.ai/api/v1",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     60 * time.Second,
		},
		Defaults: DefaultConfig{
			MaxConcurrent:  3,
			OutputMode:     "separate",
			OutputSuffix:   ".presto",
			BackupOriginal: true,
			RemoveComments: false,
			FilePattern:    ".*",
		},
		Filters: FilterConfig{
			MaxFileSize: 1024 * 1024, // 1MB
			ExcludeDirs: []string{".git", "node_modules", "vendor", ".vscode", ".idea"},
			ExcludeExts: []string{".exe", ".bin", ".so", ".dll", ".dylib"},
		},
	}
}

// LoadConfig loads configuration from file or returns defaults
func LoadConfig(path string) (*Config, error) {
	config := DefaultConfig()

	if path == "" {
		return config, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// GetAPIKey retrieves the API key from environment
func (c *AIConfig) GetAPIKey() string {
	return os.Getenv(c.APIKeyEnv)
}
