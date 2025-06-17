package types

import (
	"errors"
	"time"
)

// Language represents a programming language or file type
type Language string

const (
	LangUnknown    Language = "unknown"
	LangGo         Language = "go"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangPython     Language = "python"
	LangJava       Language = "java"
	LangC          Language = "c"
	LangCPP        Language = "cpp"
	LangRust       Language = "rust"
	LangPHP        Language = "php"
	LangRuby       Language = "ruby"
	LangShell      Language = "shell"
	LangSQL        Language = "sql"
	LangHTML       Language = "html"
	LangCSS        Language = "css"
	LangXML        Language = "xml"
	LangJSON       Language = "json"
	LangYAML       Language = "yaml"
	LangMarkdown   Language = "markdown"
	LangText       Language = "text"
)

// CommentStyle represents the style of comments for a language
type CommentStyle struct {
	LineComment string
	BlockStart  string
	BlockEnd    string
	DocComment  string
}

// ProcessingMode defines how files should be processed
type ProcessingMode string

const (
	ModeTransform ProcessingMode = "transform" // Modify existing files
	ModeGenerate  ProcessingMode = "generate"  // Create new files
)

// OutputMode defines where processed content should go
type OutputMode string

const (
	OutputModeInPlace  OutputMode = "inplace"  // Modify original files
	OutputModeSeparate OutputMode = "separate" // Create new files with suffix
	OutputModeStdout   OutputMode = "stdout"   // Print to terminal
	OutputModeFile     OutputMode = "file"     // Single output file
)

// ProcessingOptions contains all options for file processing
type ProcessingOptions struct {
	// Core options
	Mode         ProcessingMode
	AIPrompt     string
	PromptFile   string
	InputPath    string
	OutputPath   string
	OutputMode   OutputMode
	OutputSuffix string

	// File filtering
	ContextFiles    []string
	ContextPatterns []string
	Recursive       bool
	FilePattern     string
	ExcludePattern  string

	// Processing options
	RemoveComments bool
	DryRun         bool
	Verbose        bool
	MaxConcurrent  int
	BackupOriginal bool

	// AI options
	Model       string
	Temperature float64
	MaxTokens   int
}

// FileInfo represents information about a file to be processed
type FileInfo struct {
	Path         string
	OriginalPath string
	Language     Language
	Size         int64
}

// ContextFile represents a file used as context for AI processing
type ContextFile struct {
	Path     string
	Language Language
	Content  string
	Label    string
}

// AIRequest represents a request to the AI service
type AIRequest struct {
	Prompt      string
	Content     string
	Language    Language
	MaxTokens   int
	Temperature float64
	Mode        ProcessingMode
}

// AIResponse represents a response from the AI service
type AIResponse struct {
	Content    string
	TokensUsed int
	Model      string
}

// ProcessingResult represents the result of processing a file
type ProcessingResult struct {
	InputFile    string
	OutputFile   string
	Success      bool
	Skipped      bool
	SkipReason   string
	Error        error
	BytesChanged int
	AITokensUsed int
	Mode         ProcessingMode
	Duration     time.Duration
}

// Command represents a prefab command with predefined settings
type Command struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Mode        ProcessingMode    `yaml:"mode"`
	Prompt      string            `yaml:"prompt,omitempty"`
	PromptFile  string            `yaml:"prompt_file,omitempty"`
	Options     CommandOptions    `yaml:"options,omitempty"`
	Variables   map[string]string `yaml:"variables,omitempty"`
}

// CommandOptions represents options that can be set in a command
type CommandOptions struct {
	OutputMode      string   `yaml:"output_mode,omitempty"`
	OutputSuffix    string   `yaml:"output_suffix,omitempty"`
	FilePattern     string   `yaml:"file_pattern,omitempty"`
	ExcludePattern  string   `yaml:"exclude_pattern,omitempty"`
	ContextPatterns []string `yaml:"context_patterns,omitempty"`
	ContextFiles    []string `yaml:"context_files,omitempty"`
	Recursive       bool     `yaml:"recursive,omitempty"`
	RemoveComments  bool     `yaml:"remove_comments,omitempty"`
	BackupOriginal  bool     `yaml:"backup_original,omitempty"`
	Model           string   `yaml:"model,omitempty"`
	Temperature     float64  `yaml:"temperature,omitempty"`
	MaxTokens       int      `yaml:"max_tokens,omitempty"`
}

// Common errors
var (
	ErrInvalidMode       = errors.New("invalid processing mode")
	ErrInvalidOutputMode = errors.New("invalid output mode")
	ErrMissingPrompt     = errors.New("prompt or prompt file required")
	ErrMissingInput      = errors.New("input path required")
	ErrMissingOutput     = errors.New("output file required for generate mode")
)

// AIProvider represents different AI providers
type AIProvider string

const (
	ProviderOpenAI    AIProvider = "openai"
	ProviderAnthropic AIProvider = "anthropic"
	ProviderLocal     AIProvider = "local"
	ProviderCustom    AIProvider = "custom"
)

// APIConfig represents API configuration
type APIConfig struct {
	Provider    AIProvider `yaml:"provider"`
	APIKey      string     `yaml:"api_key,omitempty"`
	BaseURL     string     `yaml:"base_url"`
	Model       string     `yaml:"model"`
	MaxTokens   int        `yaml:"max_tokens"`
	Temperature float64    `yaml:"temperature"`
	Timeout     int        `yaml:"timeout_seconds"`
}

// GetDefaultConfig returns default config for a provider
func GetDefaultAPIConfig(provider AIProvider) APIConfig {
	switch provider {
	case ProviderOpenAI:
		return APIConfig{
			Provider:    ProviderOpenAI,
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     60,
		}
	case ProviderAnthropic:
		return APIConfig{
			Provider:    ProviderAnthropic,
			BaseURL:     "https://api.anthropic.com/v1",
			Model:       "claude-3-5-sonnet-20241022",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     60,
		}
	case ProviderLocal:
		return APIConfig{
			Provider:    ProviderLocal,
			BaseURL:     "http://localhost:1234/v1", // Common local API port
			Model:       "local-model",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     60,
		}
	default:
		return APIConfig{
			Provider:    ProviderCustom,
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4",
			MaxTokens:   4000,
			Temperature: 0.1,
			Timeout:     60,
		}
	}
}
