package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zachacious/presto/pkg/types"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	AI       types.APIConfig `yaml:"ai"`
	Defaults DefaultsConfig  `yaml:"defaults"`
	Filters  FiltersConfig   `yaml:"filters"`
}

// DefaultsConfig contains default processing options
type DefaultsConfig struct {
	MaxConcurrent  int    `yaml:"max_concurrent"`
	OutputMode     string `yaml:"output_mode"`
	OutputSuffix   string `yaml:"output_suffix"`
	BackupOriginal bool   `yaml:"backup_original"`
	RemoveComments bool   `yaml:"remove_comments"`
	FilePattern    string `yaml:"file_pattern"`
}

// FiltersConfig contains file filtering options
type FiltersConfig struct {
	MaxFileSize  int64    `yaml:"max_file_size"`
	ExcludeDirs  []string `yaml:"exclude_dirs"`
	ExcludeExts  []string `yaml:"exclude_exts"`
	IncludeExts  []string `yaml:"include_exts"`
	ExcludeFiles []string `yaml:"exclude_files"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		AI: types.GetDefaultAPIConfig(types.ProviderOpenAI),
		Defaults: DefaultsConfig{
			MaxConcurrent:  3,
			OutputMode:     "separate",
			OutputSuffix:   ".presto",
			BackupOriginal: true,
			RemoveComments: false,
			FilePattern:    "",
		},
		Filters: FiltersConfig{
			MaxFileSize: 1024 * 1024, // 1MB
			ExcludeDirs: []string{
				".git", ".svn", ".hg",
				"node_modules", "vendor", "venv", ".venv",
				".idea", ".vscode", ".vs",
				"dist", "build", "target", "bin", "obj",
				".next", ".nuxt", "__pycache__",
				"coverage", ".coverage", ".nyc_output",
			},
			ExcludeExts: []string{
				".exe", ".dll", ".so", ".dylib",
				".bin", ".o", ".obj", ".a", ".lib",
				".zip", ".tar", ".gz", ".rar",
				".jpg", ".jpeg", ".png", ".gif", ".bmp",
				".mp3", ".mp4", ".avi", ".mov", ".wmv",
				".pdf", ".doc", ".docx", ".xls", ".xlsx",
			},
			IncludeExts:  []string{},
			ExcludeFiles: []string{"*.min.*", "*.bundle.*", "*-lock.*"},
		},
	}
}

// LoadConfig loads configuration from file with environment variable fallbacks
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Determine config file path
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return cfg, nil
		}
		configPath = filepath.Join(homeDir, ".presto", "config.yaml")
	}

	// Load from file if it exists
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Apply environment variable overrides
	cfg.applyEnvOverrides()

	return cfg, nil
}

// applyEnvOverrides applies environment variable overrides
func (c *Config) applyEnvOverrides() {
	// Try multiple environment variable names for compatibility
	envVars := []string{
		"OPENAI_API_KEY",     // OpenAI standard
		"ANTHROPIC_API_KEY",  // Anthropic standard
		"OPENROUTER_API_KEY", // OpenRouter (for backward compatibility)
		"AI_API_KEY",         // Generic
		"PRESTO_API_KEY",     // Presto-specific
	}

	for _, envVar := range envVars {
		if key := os.Getenv(envVar); key != "" && c.AI.APIKey == "" {
			c.AI.APIKey = key
			break
		}
	}

	// Override other settings from env if present
	if baseURL := os.Getenv("PRESTO_BASE_URL"); baseURL != "" {
		c.AI.BaseURL = baseURL
	}
	if model := os.Getenv("PRESTO_MODEL"); model != "" {
		c.AI.Model = model
	}
}

// SaveConfig saves configuration to file
func SaveConfig(cfg *Config, configPath string) error {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(homeDir, ".presto", "config.yaml")
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// SaveAPIKey saves API key to config file
func SaveAPIKey(apiKey string) error {
	cfg, err := LoadConfig("")
	if err != nil {
		cfg = DefaultConfig()
	}

	cfg.AI.APIKey = apiKey
	return SaveConfig(cfg, "")
}

// PromptForAPIKey prompts user to enter API key interactively
func PromptForAPIKey() (string, error) {
	fmt.Print("🔑 API key not found.\n")
	fmt.Print("Please enter your API key: ")

	var apiKey string
	if _, err := fmt.Scanln(&apiKey); err != nil {
		return "", fmt.Errorf("failed to read API key: %w", err)
	}

	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return apiKey, nil
}

// ConfigureInteractive runs interactive configuration
func ConfigureInteractive() error {
	fmt.Println("🎩 Presto Configuration")
	fmt.Println("=======================")
	fmt.Println()

	// Choose provider
	provider, err := promptProvider()
	if err != nil {
		return err
	}

	// Get base config for provider
	cfg := DefaultConfig()
	cfg.AI = types.GetDefaultAPIConfig(provider)

	// Configure API key
	if err := configureAPIKey(cfg, provider); err != nil {
		return err
	}

	// Configure model (optional)
	if err := configureModel(cfg, provider); err != nil {
		return err
	}

	// Save configuration
	if err := SaveConfig(cfg, ""); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Configuration saved to ~/.presto/config.yaml")
	fmt.Println()
	fmt.Println("You can now run commands like:")
	fmt.Println("  presto --cmd add-docs --input .")
	fmt.Println("  presto --prompt \"Add comments\" --input main.go")

	return nil
}

func promptProvider() (types.AIProvider, error) {
	fmt.Println("Select AI provider:")
	fmt.Println("1. OpenAI (gpt-4, gpt-3.5-turbo)")
	fmt.Println("2. Anthropic (claude-3-5-sonnet, claude-3-haiku)")
	fmt.Println("3. Local (ollama, lm-studio, etc.)")
	fmt.Println("4. Custom (other OpenAI-compatible API)")
	fmt.Print("Choice [1]: ")

	var choice string
	fmt.Scanln(&choice)
	choice = strings.TrimSpace(choice)

	if choice == "" {
		choice = "1"
	}

	switch choice {
	case "1":
		return types.ProviderOpenAI, nil
	case "2":
		return types.ProviderAnthropic, nil
	case "3":
		return types.ProviderLocal, nil
	case "4":
		return types.ProviderCustom, nil
	default:
		return types.ProviderOpenAI, nil
	}
}

func configureAPIKey(cfg *Config, provider types.AIProvider) error {
	switch provider {
	case types.ProviderOpenAI:
		fmt.Println()
		fmt.Println("🔑 OpenAI API Key")
		fmt.Println("Get your API key from: https://platform.openai.com/api-keys")
		fmt.Print("Enter API key: ")
	case types.ProviderAnthropic:
		fmt.Println()
		fmt.Println("🔑 Anthropic API Key")
		fmt.Println("Get your API key from: https://console.anthropic.com/")
		fmt.Print("Enter API key: ")
	case types.ProviderLocal:
		fmt.Println()
		fmt.Println("🔑 Local API Configuration")
		fmt.Println("For local models, you may not need an API key.")
		fmt.Print("Enter API key (or press Enter to skip): ")
	case types.ProviderCustom:
		fmt.Println()
		fmt.Println("🔑 Custom API Configuration")
		fmt.Print("Enter base URL: ")
		var baseURL string
		fmt.Scanln(&baseURL)
		if baseURL != "" {
			cfg.AI.BaseURL = baseURL
		}
		fmt.Print("Enter API key: ")
	}

	var apiKey string
	fmt.Scanln(&apiKey)
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" && provider != types.ProviderLocal {
		return fmt.Errorf("API key is required for %s", provider)
	}

	cfg.AI.APIKey = apiKey
	return nil
}

func configureModel(cfg *Config, provider types.AIProvider) error {
	fmt.Println()
	fmt.Printf("Default model: %s\n", cfg.AI.Model)

	switch provider {
	case types.ProviderOpenAI:
		fmt.Println("Available models: gpt-4, gpt-4-turbo, gpt-3.5-turbo")
	case types.ProviderAnthropic:
		fmt.Println("Available models: claude-3-5-sonnet-20241022, claude-3-haiku-20240307")
	case types.ProviderLocal:
		fmt.Println("Use the model name from your local setup")
	}

	fmt.Print("Enter model name (or press Enter for default): ")
	var model string
	fmt.Scanln(&model)
	model = strings.TrimSpace(model)

	if model != "" {
		cfg.AI.Model = model
	}

	return nil
}

// ValidateConfig checks if configuration is valid
func ValidateConfig(cfg *Config) error {
	if cfg.AI.APIKey == "" && cfg.AI.Provider != types.ProviderLocal {
		return fmt.Errorf("API key is required")
	}
	if cfg.AI.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if cfg.AI.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".presto", "config.yaml")
}
