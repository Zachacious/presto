package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Zachacious/presto/pkg/types"
)

// Client handles AI API requests
type Client struct {
	config     *types.APIConfig
	httpClient *http.Client
}

// New creates a new AI client
func New(cfg *types.APIConfig) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
	}
}

// ValidateConfig checks if the AI configuration is valid
func (c *Client) ValidateConfig() error {
	if c.config.APIKey == "" && c.config.Provider != types.ProviderLocal {
		return fmt.Errorf("API key is required for provider %s", c.config.Provider)
	}
	if c.config.BaseURL == "" {
		return fmt.Errorf("base URL is required")
	}
	if c.config.Model == "" {
		return fmt.Errorf("model is required")
	}
	return nil
}

// ProcessContent sends content to AI for processing
func (c *Client) ProcessContent(req types.AIRequest, contextFiles []*types.ContextFile) (*types.AIResponse, error) {
	// Build the complete prompt
	fullPrompt := c.buildPrompt(req, contextFiles)

	// Create API request based on provider
	var apiResp APIResponse
	var err error

	switch c.config.Provider {
	case types.ProviderOpenAI, types.ProviderLocal, types.ProviderCustom:
		apiResp, err = c.sendOpenAIRequest(fullPrompt, req)
	case types.ProviderAnthropic:
		apiResp, err = c.sendAnthropicRequest(fullPrompt, req)
	default:
		// Default to OpenAI-compatible API
		apiResp, err = c.sendOpenAIRequest(fullPrompt, req)
	}

	if err != nil {
		return nil, err
	}

	content := apiResp.GetContent()

	// Post-process the content to remove unwanted markdown formatting
	if req.Mode == types.ModeTransform {
		content = c.postProcessContent(content, req.Language)
	}

	// Convert to standard response
	return &types.AIResponse{
		Content:    content,
		TokensUsed: apiResp.GetTokensUsed(),
		Model:      c.config.Model,
	}, nil
}

// postProcessContent removes unwanted markdown formatting from AI responses
func (c *Client) postProcessContent(content string, language types.Language) string {
	// Don't process markdown files - they should keep their code blocks
	if language == types.LangMarkdown {
		return content
	}

	// Remove outer code block wrapping if present
	return c.removeOuterCodeBlock(content, language)
}

// removeOuterCodeBlock safely removes outer markdown code block wrapping
func (c *Client) removeOuterCodeBlock(content string, language types.Language) string {
	content = strings.TrimSpace(content)

	if content == "" {
		return content
	}

	// Pattern to match outer code blocks: ```language\ncontent\n```
	// We need to be very careful to only match the outermost wrapper

	// Check if content starts with ``` and ends with ```
	if !strings.HasPrefix(content, "```") || !strings.HasSuffix(content, "```") {
		return content // No code block wrapping
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return content // Too short to be a wrapped code block
	}

	firstLine := strings.TrimSpace(lines[0])
	lastLine := strings.TrimSpace(lines[len(lines)-1])

	// First line should be ``` optionally followed by language
	if !strings.HasPrefix(firstLine, "```") {
		return content
	}

	// Last line should be exactly ```
	if lastLine != "```" {
		return content
	}

	// Extract the language from the first line
	declaredLang := strings.TrimSpace(strings.TrimPrefix(firstLine, "```"))

	// Check if the declared language makes sense for our file
	if declaredLang != "" && !c.isCompatibleLanguage(declaredLang, language) {
		// Language mismatch - might not be a wrapper, could be actual content
		return content
	}

	// Extract content between the outer code blocks
	innerContent := strings.Join(lines[1:len(lines)-1], "\n")

	// Additional safety check: make sure we're not removing content that has
	// its own code blocks (which would indicate this isn't just a wrapper)
	if c.hasInnerCodeBlocks(innerContent) {
		// This looks like actual markdown content with code blocks inside
		// Only remove if we're confident this is a wrapper
		if c.looksLikeWrapper(content, innerContent, language) {
			return innerContent
		}
		return content
	}

	return innerContent
}

// isCompatibleLanguage checks if the declared language is compatible with the file language
func (c *Client) isCompatibleLanguage(declared string, fileLanguage types.Language) bool {
	declared = strings.ToLower(declared)

	languageAliases := map[string][]string{
		"go":         {"go", "golang"},
		"javascript": {"js", "javascript", "jsx"},
		"typescript": {"ts", "typescript", "tsx"},
		"python":     {"py", "python", "python3"},
		"rust":       {"rs", "rust"},
		"java":       {"java"},
		"cpp":        {"cpp", "c++", "cxx"},
		"c":          {"c"},
		"csharp":     {"cs", "csharp", "c#"},
		"php":        {"php"},
		"ruby":       {"rb", "ruby"},
		"yaml":       {"yml", "yaml"},
		"json":       {"json"},
		"xml":        {"xml"},
		"html":       {"html", "htm"},
		"css":        {"css"},
		"sql":        {"sql"},
		"shell":      {"sh", "bash", "shell", "zsh"},
	}

	if aliases, exists := languageAliases[string(fileLanguage)]; exists {
		for _, alias := range aliases {
			if declared == alias {
				return true
			}
		}
	}

	return declared == string(fileLanguage)
}

// hasInnerCodeBlocks checks if content contains code blocks (indicating it might be actual markdown)
func (c *Client) hasInnerCodeBlocks(content string) bool {
	// Count occurrences of ```
	count := strings.Count(content, "```")
	// If there are code blocks inside, there should be at least 2 ``` (open and close)
	return count >= 2
}

// looksLikeWrapper determines if this looks like an AI wrapper vs actual content
func (c *Client) looksLikeWrapper(fullContent, innerContent string, language types.Language) bool {
	// If the inner content looks like valid code/file content for our language,
	// and the full content is just that plus a wrapper, it's probably a wrapper

	// Check the ratio - if inner content is most of the full content, likely a wrapper
	fullLines := len(strings.Split(fullContent, "\n"))
	innerLines := len(strings.Split(innerContent, "\n"))

	// If we only added 2 lines (opening and closing ```), it's likely a wrapper
	if fullLines-innerLines <= 2 {
		return true
	}

	// For code files, check if inner content starts with typical code patterns
	switch language {
	case types.LangGo:
		return strings.HasPrefix(strings.TrimSpace(innerContent), "package ") ||
			strings.Contains(innerContent, "func ") ||
			strings.Contains(innerContent, "import ")
	case types.LangJavaScript, types.LangTypeScript:
		return strings.Contains(innerContent, "function ") ||
			strings.Contains(innerContent, "const ") ||
			strings.Contains(innerContent, "import ") ||
			strings.Contains(innerContent, "export ")
	case types.LangPython:
		return strings.Contains(innerContent, "def ") ||
			strings.Contains(innerContent, "class ") ||
			strings.Contains(innerContent, "import ")
	}

	// For other file types, be conservative
	return fullLines-innerLines <= 2
}

// Update buildPrompt to include stronger plaintext instructions
func (c *Client) buildPrompt(req types.AIRequest, contextFiles []*types.ContextFile) string {
	var prompt bytes.Buffer

	// Add current file context first (if we're transforming a specific file)
	if req.Mode == types.ModeTransform && req.FileName != "" {
		prompt.WriteString("Current file being processed:\n")
		prompt.WriteString(c.gatherFileContext(req.FileName))
		prompt.WriteString("\n")
	}

	// Add context files if provided
	if len(contextFiles) > 0 {
		prompt.WriteString("Context files:\n\n")
		for _, file := range contextFiles {
			prompt.WriteString(fmt.Sprintf("=== %s (%s) ===\n", file.Label, file.Language))
			prompt.WriteString(file.Content)
			prompt.WriteString("\n\n")
		}
		prompt.WriteString("---\n\n")
	}

	// Add the main prompt
	prompt.WriteString(req.Prompt)

	// Add target content if transforming
	if req.Mode == types.ModeTransform && req.Content != "" {
		prompt.WriteString("\n\nContent to transform:\n\n")
		prompt.WriteString(req.Content)
	}

	// Add explicit instructions to prevent markdown formatting
	if req.Mode == types.ModeTransform {
		prompt.WriteString("\n\n" + c.getOutputInstructions(req.Language))
	}

	return prompt.String()
}

// getOutputInstructions returns language-specific output instructions
func (c *Client) getOutputInstructions(language types.Language) string {
	var instructions bytes.Buffer

	instructions.WriteString("=== OUTPUT INSTRUCTIONS ===\n")

	if language == types.LangMarkdown {
		instructions.WriteString("Return the content as properly formatted Markdown.\n")
		instructions.WriteString("Preserve all existing code blocks and markdown formatting.\n")
	} else {
		instructions.WriteString("CRITICAL: Return ONLY the raw file content in plain text format.\n")
		instructions.WriteString("Do NOT wrap the output in markdown code blocks (```).\n")
		instructions.WriteString("Do NOT include any explanatory text before or after the code.\n")
		instructions.WriteString("Do NOT format as markdown - return the exact file content.\n")
		instructions.WriteString(fmt.Sprintf("The output should be valid %s code that can be saved directly to a file.\n", language))
	}

	instructions.WriteString("Do not include any commentary, explanations, or wrapper text.\n")
	instructions.WriteString("Start your response immediately with the file content.")

	return instructions.String()
}

// gatherFileContext collects useful context about the current file and environment
func (c *Client) gatherFileContext(fileName string) string {
	var context bytes.Buffer

	// Current file info
	context.WriteString(fmt.Sprintf("File: %s\n", fileName))

	// File stats
	if info, err := os.Stat(fileName); err == nil {
		context.WriteString(fmt.Sprintf("Size: %d bytes\n", info.Size()))
		context.WriteString(fmt.Sprintf("Last modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05")))
		context.WriteString(fmt.Sprintf("File mode: %s\n", info.Mode().String()))
	}

	// Current timestamp and timezone
	now := time.Now()
	context.WriteString(fmt.Sprintf("Processing time: %s\n", now.Format("2006-01-02 15:04:05 MST")))
	context.WriteString(fmt.Sprintf("Timestamp: %d\n", now.Unix()))

	// File path context
	absPath, _ := filepath.Abs(fileName)
	context.WriteString(fmt.Sprintf("Absolute path: %s\n", absPath))
	context.WriteString(fmt.Sprintf("Directory: %s\n", filepath.Dir(absPath)))
	context.WriteString(fmt.Sprintf("Base name: %s\n", filepath.Base(fileName)))
	context.WriteString(fmt.Sprintf("Extension: %s\n", filepath.Ext(fileName)))

	// Working directory
	if wd, err := os.Getwd(); err == nil {
		context.WriteString(fmt.Sprintf("Working directory: %s\n", wd))

		// Check if we're in a git repository
		if gitRoot := findGitRoot(wd); gitRoot != "" {
			context.WriteString(fmt.Sprintf("Git repository: %s\n", gitRoot))
			relPath, _ := filepath.Rel(gitRoot, absPath)
			context.WriteString(fmt.Sprintf("Path from git root: %s\n", relPath))
		}

		// Check for common project files
		projectContext := detectProjectContext(wd)
		if projectContext != "" {
			context.WriteString(fmt.Sprintf("Project type: %s\n", projectContext))
		}
	}

	// System context
	context.WriteString(fmt.Sprintf("OS: %s\n", runtime.GOOS))
	context.WriteString(fmt.Sprintf("Architecture: %s\n", runtime.GOARCH))

	// User context (if available)
	if currentUser, err := user.Current(); err == nil {
		context.WriteString(fmt.Sprintf("User: %s\n", currentUser.Username))
	}

	// Environment variables that might be relevant
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		context.WriteString(fmt.Sprintf("GOPATH: %s\n", gopath))
	}
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		context.WriteString(fmt.Sprintf("GOROOT: %s\n", goroot))
	}
	if nodeEnv := os.Getenv("NODE_ENV"); nodeEnv != "" {
		context.WriteString(fmt.Sprintf("NODE_ENV: %s\n", nodeEnv))
	}

	return context.String()
}

// findGitRoot walks up the directory tree to find the git repository root
func findGitRoot(startPath string) string {
	path := startPath
	for {
		gitDir := filepath.Join(path, ".git")
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			return path
		}

		parent := filepath.Dir(path)
		if parent == path {
			break // reached filesystem root
		}
		path = parent
	}
	return ""
}

// detectProjectContext tries to detect what kind of project this is
func detectProjectContext(dir string) string {
	var projectTypes []string

	// Check for common project files
	projectFiles := map[string]string{
		"package.json":       "Node.js",
		"go.mod":             "Go module",
		"Cargo.toml":         "Rust",
		"pom.xml":            "Maven/Java",
		"build.gradle":       "Gradle/Java",
		"requirements.txt":   "Python",
		"Pipfile":            "Python/Pipenv",
		"composer.json":      "PHP/Composer",
		"Gemfile":            "Ruby/Bundler",
		"Dockerfile":         "Docker",
		"docker-compose.yml": "Docker Compose",
		"Makefile":           "Make",
		"CMakeLists.txt":     "CMake",
	}

	for file, projectType := range projectFiles {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			projectTypes = append(projectTypes, projectType)
		}
	}

	return strings.Join(projectTypes, ", ")
}

// sendOpenAIRequest sends request to OpenAI-compatible API
func (c *Client) sendOpenAIRequest(prompt string, req types.AIRequest) (APIResponse, error) {
	// Build request
	openAIReq := OpenAIRequest{
		Model: c.config.Model,
		Messages: []OpenAIMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   c.getMaxTokens(req.MaxTokens),
		Temperature: c.getTemperature(req.Temperature),
	}

	// Marshal request
	jsonData, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", c.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}
	httpReq.Header.Set("User-Agent", "Presto/1.0")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}

// sendAnthropicRequest sends request to Anthropic API
func (c *Client) sendAnthropicRequest(prompt string, req types.AIRequest) (APIResponse, error) {
	// Build request
	anthropicReq := AnthropicRequest{
		Model:       c.config.Model,
		MaxTokens:   c.getMaxTokens(req.MaxTokens),
		Temperature: c.getTemperature(req.Temperature),
		Messages: []AnthropicMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Marshal request
	jsonData, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", c.config.BaseURL+"/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.config.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d", resp.StatusCode)
	}

	// Parse response
	var apiResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &apiResp, nil
}

// Helper methods
func (c *Client) getMaxTokens(requestTokens int) int {
	if requestTokens > 0 {
		return requestTokens
	}
	return c.config.MaxTokens
}

func (c *Client) getTemperature(requestTemp float64) float64 {
	if requestTemp > 0 {
		return requestTemp
	}
	return c.config.Temperature
}

// API Types

// APIResponse interface for different provider responses
type APIResponse interface {
	GetContent() string
	GetTokensUsed() int
}

// OpenAI API types
type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Message OpenAIMessage `json:"message"`
}

type OpenAIUsage struct {
	TotalTokens int `json:"total_tokens"`
}

func (r *OpenAIResponse) GetContent() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Content
	}
	return ""
}

func (r *OpenAIResponse) GetTokensUsed() int {
	return r.Usage.TotalTokens
}

// Anthropic API types
type AnthropicRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Messages    []AnthropicMessage `json:"messages"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	Content []AnthropicContent `json:"content"`
	Usage   AnthropicUsage     `json:"usage"`
}

type AnthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func (r *AnthropicResponse) GetContent() string {
	if len(r.Content) > 0 && r.Content[0].Type == "text" {
		return r.Content[0].Text
	}
	return ""
}

func (r *AnthropicResponse) GetTokensUsed() int {
	return r.Usage.InputTokens + r.Usage.OutputTokens
}
