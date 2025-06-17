package types

import "time"

// ProcessingMode defines how presto should operate
type ProcessingMode string

const (
	ModeTransform ProcessingMode = "transform" // Transform existing files
	ModeGenerate  ProcessingMode = "generate"  // Generate new file from context
)

// ProcessingOptions defines how files should be processed
type ProcessingOptions struct {
	Mode            ProcessingMode
	AIPrompt        string
	PromptFile      string
	InputPath       string
	OutputPath      string
	OutputMode      OutputMode
	ContextPatterns []string // NEW: Patterns for context files
	ContextFiles    []string // Specific context files
	Recursive       bool
	FilePattern     string
	ExcludePattern  string
	RemoveComments  bool
	DryRun          bool
	Verbose         bool
	MaxConcurrent   int
	BackupOriginal  bool
	OutputSuffix    string
	Model           string
	Temperature     float64
	MaxTokens       int
}

type OutputMode string

const (
	OutputModeInPlace  OutputMode = "inplace"
	OutputModeSeparate OutputMode = "separate"
	OutputModeStdout   OutputMode = "stdout"
	OutputModeFile     OutputMode = "file" // NEW: Single output file
)

// Command represents a prefab command
type Command struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Mode        ProcessingMode    `yaml:"mode"`
	Prompt      string            `yaml:"prompt,omitempty"`
	PromptFile  string            `yaml:"prompt_file,omitempty"`
	Options     CommandOptions    `yaml:"options"`
	Variables   map[string]string `yaml:"variables,omitempty"`
}

// CommandOptions holds all the command line options
type CommandOptions struct {
	OutputMode      string   `yaml:"output_mode,omitempty"`
	OutputSuffix    string   `yaml:"output_suffix,omitempty"`
	Recursive       bool     `yaml:"recursive,omitempty"`
	FilePattern     string   `yaml:"file_pattern,omitempty"`
	ExcludePattern  string   `yaml:"exclude_pattern,omitempty"`
	ContextPatterns []string `yaml:"context_patterns,omitempty"`
	ContextFiles    []string `yaml:"context_files,omitempty"`
	RemoveComments  bool     `yaml:"remove_comments,omitempty"`
	BackupOriginal  bool     `yaml:"backup_original,omitempty"`
	Model           string   `yaml:"model,omitempty"`
	Temperature     float64  `yaml:"temperature,omitempty"`
	MaxTokens       int      `yaml:"max_tokens,omitempty"`
}

// ContextFile represents a file used for context
type ContextFile struct {
	Path     string
	Language Language
	Content  string
	Label    string
}

// FileInfo represents a file to be processed
type FileInfo struct {
	Path         string
	OriginalPath string
	Language     Language
	Size         int64
	Content      string
}

// ProcessingResult represents the result of processing
type ProcessingResult struct {
	InputFile    string
	OutputFile   string
	Success      bool
	Error        error
	BytesChanged int
	Duration     time.Duration
	AITokensUsed int
	Mode         ProcessingMode
}

// Language represents programming language or file type
type Language string

const (
	LangGo         Language = "go"
	LangJavaScript Language = "javascript"
	LangTypeScript Language = "typescript"
	LangPython     Language = "python"
	LangJava       Language = "java"
	LangC          Language = "c"
	LangCPP        Language = "cpp"
	LangRust       Language = "rust"
	LangHTML       Language = "html"
	LangCSS        Language = "css"
	LangSQL        Language = "sql"
	LangJSON       Language = "json"
	LangYAML       Language = "yaml"
	LangMarkdown   Language = "markdown"
	LangShell      Language = "shell"
	LangText       Language = "text"
	LangUnknown    Language = "unknown"
)

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
	Duration   time.Duration
}
