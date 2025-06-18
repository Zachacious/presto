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
	const MAX_CONTINUATIONS = 5

	var fullContent strings.Builder
	var totalTokens int
	var lastFinishReason string
	currentPrompt := c.buildPrompt(req, contextFiles)

	for attempt := 0; attempt < MAX_CONTINUATIONS; attempt++ {
		// Create API request based on provider
		var apiResp APIResponse
		var err error

		switch c.config.Provider {
		case types.ProviderOpenAI, types.ProviderLocal, types.ProviderCustom:
			apiResp, err = c.sendOpenAIRequest(currentPrompt, req)
		case types.ProviderAnthropic:
			apiResp, err = c.sendAnthropicRequest(currentPrompt, req)
		default:
			apiResp, err = c.sendOpenAIRequest(currentPrompt, req)
		}

		if err != nil {
			return nil, err
		}

		content := apiResp.GetContent()
		totalTokens += apiResp.GetTokensUsed()
		lastFinishReason = apiResp.GetFinishReason()

		// Post-process the content to remove unwanted markdown formatting
		if req.Mode == types.ModeTransform {
			content = c.postProcessContent(content, req.Language)
		}

		// First response - add everything
		if attempt == 0 {
			fullContent.WriteString(content)
		} else {
			// Continuation - try to merge intelligently
			merged := c.MergeContinuation(fullContent.String(), content)
			fullContent.Reset()
			fullContent.WriteString(merged)
		}

		// Check if response is complete
		if apiResp.IsComplete() {
			break
		}

		// For generate mode, don't continue (it's not a file completion)
		if req.Mode == types.ModeGenerate {
			break
		}

		// Check if we have reasonable completeness
		if attempt > 0 && c.LooksReasonablyComplete(fullContent.String(), req.Content, req.Language) {
			break
		}

		// Prepare continuation prompt
		currentPrompt = c.BuildContinuationPrompt(fullContent.String(), req.Content, req)
	}

	// Convert to standard response
	return &types.AIResponse{
		Content:      fullContent.String(),
		TokensUsed:   totalTokens,
		Model:        c.config.Model,
		FinishReason: lastFinishReason,
		Truncated:    !c.wasResponseComplete(lastFinishReason),
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

// NEW: Build continuation prompt
func (c *Client) BuildContinuationPrompt(partialContent, originalContent string, req types.AIRequest) string {
	var prompt bytes.Buffer

	prompt.WriteString("CONTINUATION REQUEST:\n")
	prompt.WriteString("The previous response was incomplete due to length limits. Please continue from where you left off.\n\n")

	prompt.WriteString("IMPORTANT INSTRUCTIONS:\n")
	prompt.WriteString("- Continue the file content exactly from where it was cut off\n")
	prompt.WriteString("- Return ONLY the remaining content - no explanations or repetition\n")
	prompt.WriteString("- Do not repeat any content that was already provided\n")
	prompt.WriteString("- Maintain the same formatting and style\n")
	prompt.WriteString("- Complete the entire file\n\n")

	// Show original file size for context
	if originalContent != "" {
		prompt.WriteString(fmt.Sprintf("ORIGINAL FILE SIZE: %d characters\n", len(originalContent)))
		prompt.WriteString(fmt.Sprintf("PARTIAL CONTENT SIZE: %d characters\n", len(partialContent)))
		prompt.WriteString(fmt.Sprintf("ESTIMATED REMAINING: %d characters\n\n", len(originalContent)-len(partialContent)))
	}

	// Show the last few lines of what we have so far for context
	lines := strings.Split(partialContent, "\n")
	contextLines := 10
	if len(lines) > contextLines {
		prompt.WriteString("LAST FEW LINES OF PARTIAL CONTENT (for context):\n")
		prompt.WriteString("...\n")
		prompt.WriteString(strings.Join(lines[len(lines)-contextLines:], "\n"))
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please continue with the remaining content to complete the file.\n")
	prompt.WriteString(c.getOutputInstructions(req.Language))

	return prompt.String()
}

// Enhanced mergeContinuation with robust overlap detection and removal
func (c *Client) MergeContinuation(existing, continuation string) string {
	continuation = strings.TrimSpace(continuation)
	if continuation == "" {
		return existing
	}

	// First try exact line overlap detection (existing logic)
	if merged := c.tryExactLineOverlap(existing, continuation); merged != "" {
		return merged
	}

	// Then try partial completion detection (new logic for your case)
	if merged := c.tryPartialCompletion(existing, continuation); merged != "" {
		return merged
	}

	// Finally try word-level overlap detection
	if merged := c.tryWordOverlap(existing, continuation); merged != "" {
		return merged
	}

	// No overlap detected, concatenate with newline
	return existing + "\n" + continuation
}

// tryExactLineOverlap - existing logic for exact line matches
func (c *Client) tryExactLineOverlap(existing, continuation string) string {
	existingLines := strings.Split(existing, "\n")
	continuationLines := strings.Split(continuation, "\n")

	maxOverlapCheck := min(len(existingLines), len(continuationLines), 5)

	for overlap := 1; overlap <= maxOverlapCheck; overlap++ {
		existingTail := existingLines[len(existingLines)-overlap:]
		continuationHead := continuationLines[:overlap]

		if c.linesMatch(existingTail, continuationHead) {
			remaining := continuationLines[overlap:]
			if len(remaining) > 0 {
				return existing + "\n" + strings.Join(remaining, "\n")
			}
			return existing
		}
	}

	return "" // No exact overlap found
}

// tryPartialCompletion - NEW: detect when continuation completes truncated content
func (c *Client) tryPartialCompletion(existing, continuation string) string {
	existingLines := strings.Split(existing, "\n")
	continuationLines := strings.Split(continuation, "\n")

	if len(existingLines) == 0 || len(continuationLines) == 0 {
		return ""
	}

	// Check last 1-3 lines of existing against first 1-3 lines of continuation
	maxLinesToCheck := min(len(existingLines), len(continuationLines), 3)

	for linesToRemove := 1; linesToRemove <= maxLinesToCheck; linesToRemove++ {
		// Lines to potentially remove from end of existing
		existingTail := existingLines[len(existingLines)-linesToRemove:]

		// Lines to check in continuation
		continuationHead := continuationLines[:min(linesToRemove+2, len(continuationLines), 20)]

		// Check if any line in continuation head "completes" the tail
		if completionPoint := c.findCompletionPoint(existingTail, continuationHead); completionPoint >= 0 {
			// Remove incomplete lines from existing
			truncatedExisting := strings.Join(existingLines[:len(existingLines)-linesToRemove], "\n")

			// Take continuation from the completion point
			remainingContinuation := strings.Join(continuationLines[completionPoint:], "\n")

			if truncatedExisting == "" {
				return remainingContinuation
			}
			return truncatedExisting + "\n" + remainingContinuation
		}
	}

	return "" // No partial completion found
}

// findCompletionPoint - check if continuation lines complete/correct existing lines
func (c *Client) findCompletionPoint(existingTail, continuationHead []string) int {
	for contIdx, contLine := range continuationHead {
		contLine = strings.TrimSpace(contLine)
		if contLine == "" {
			continue
		}

		// Check if this continuation line completes any existing tail line
		for _, existingLine := range existingTail {
			existingLine = strings.TrimSpace(existingLine)
			if existingLine == "" {
				continue
			}

			// Case 1: Continuation line starts with existing line content (completion)
			// existing: "He walked dow"  continuation: "He walked down the street"
			if len(contLine) > len(existingLine) && strings.HasPrefix(contLine, existingLine) {
				return contIdx
			}

			// Case 2: Existing line is prefix of continuation (truncation completion)
			// existing: "console.log('Hello"  continuation: "console.log('Hello World')"
			if len(existingLine) > 3 && strings.HasPrefix(contLine, existingLine[:len(existingLine)-1]) {
				return contIdx
			}

			// Case 3: Word-level completion
			existingWords := strings.Fields(existingLine)
			contWords := strings.Fields(contLine)

			if len(existingWords) > 0 && len(contWords) > 0 {
				lastExistingWord := existingWords[len(existingWords)-1]
				firstContWord := contWords[0]

				// Check if last word in existing is partial of first word in continuation
				if len(lastExistingWord) >= 3 && len(firstContWord) > len(lastExistingWord) &&
					strings.HasPrefix(firstContWord, lastExistingWord) {
					return contIdx
				}
			}
		}
	}

	return -1 // No completion found
}

// tryWordOverlap - detect word-level overlaps for more complex cases
func (c *Client) tryWordOverlap(existing, continuation string) string {
	// Get last few sentences of existing
	existingWords := strings.Fields(existing)
	continuationWords := strings.Fields(continuation)

	if len(existingWords) < 3 || len(continuationWords) < 3 {
		return ""
	}

	// Check last 10-20 words of existing against first 10-20 words of continuation
	maxWordsToCheck := min(len(existingWords), len(continuationWords), 20)

	for wordsToCheck := 3; wordsToCheck <= maxWordsToCheck; wordsToCheck++ {
		existingTail := existingWords[len(existingWords)-wordsToCheck:]
		continuationHead := continuationWords[:min(wordsToCheck+5, len(continuationWords), 20)]

		// Look for the longest matching sequence
		if overlapLen := c.findWordOverlap(existingTail, continuationHead); overlapLen > 0 {
			// Remove overlapping words from existing
			truncatedWords := existingWords[:len(existingWords)-overlapLen]

			// Reconstruct the text
			truncatedExisting := strings.Join(truncatedWords, " ")

			if truncatedExisting == "" {
				return continuation
			}
			return truncatedExisting + " " + continuation
		}
	}

	return "" // No word overlap found
}

// findWordOverlap - find longest word sequence overlap
func (c *Client) findWordOverlap(existingTail, continuationHead []string) int {
	maxOverlap := 0

	// Try different starting positions in continuation
	for startPos := 0; startPos < len(continuationHead)-2; startPos++ {
		overlap := 0

		// Count matching words
		for i := 0; i < len(existingTail) && (startPos+i) < len(continuationHead); i++ {
			existing := strings.ToLower(strings.TrimSpace(existingTail[i]))
			continuation := strings.ToLower(strings.TrimSpace(continuationHead[startPos+i]))

			if existing == continuation {
				overlap++
			} else {
				break
			}
		}

		// Need at least 3 matching words to consider it an overlap
		if overlap >= 3 && overlap > maxOverlap {
			maxOverlap = overlap
		}
	}

	return maxOverlap
}

// linesMatch - existing helper (unchanged)
func (c *Client) linesMatch(lines1, lines2 []string) bool {
	if len(lines1) != len(lines2) {
		return false
	}

	for i := range lines1 {
		if strings.TrimSpace(lines1[i]) != strings.TrimSpace(lines2[i]) {
			return false
		}
	}

	return true
}

// func min(a, b, c int) int {
// 	if a <= b && a <= c {
// 		return a
// 	}
// 	if b <= c {
// 		return b
// 	}
// 	return c
// }

// NEW: Check if two line slices match (allowing for minor whitespace differences)
// func (c *Client) linesMatch(lines1, lines2 []string) bool {
// 	if len(lines1) != len(lines2) {
// 		return false
// 	}

// 	for i := range lines1 {
// 		if strings.TrimSpace(lines1[i]) != strings.TrimSpace(lines2[i]) {
// 			return false
// 		}
// 	}

// 	return true
// }

// NEW: Check if response looks reasonably complete
func (c *Client) LooksReasonablyComplete(response, original string, language types.Language) bool {
	// If response is longer than original, probably complete
	if len(response) >= len(original) {
		return true
	}

	// If response is at least 90% of original and structurally valid, probably complete
	if len(response) >= len(original)*9/10 {
		if c.hasBalancedStructure(response, language) {
			return true
		}
	}

	return false
}

// NEW: Check if response has balanced structure (basic validation)
func (c *Client) hasBalancedStructure(content string, language types.Language) bool {
	switch language {
	case types.LangJavaScript, types.LangTypeScript, types.LangJSON, types.LangGo, types.LangJava, types.LangC, types.LangCPP:
		return c.hasBalancedBraces(content)
	case types.LangHTML, types.LangXML:
		return c.hasBalancedTags(content)
	case types.LangPython:
		return c.hasValidPythonStructure(content)
	default:
		// For other languages, do basic checks
		return !c.hasAbruptEnding(content)
	}
}

// NEW: Check brace balance
func (c *Client) hasBalancedBraces(content string) bool {
	braceCount := 0
	parenCount := 0
	bracketCount := 0

	inString := false
	var stringChar rune

	for i, r := range content {
		// Handle string literals (basic)
		if !inString && (r == '"' || r == '\'' || r == '`') {
			inString = true
			stringChar = r
			continue
		} else if inString && r == stringChar {
			// Check if it's escaped
			if i > 0 && content[i-1] != '\\' {
				inString = false
			}
			continue
		}

		if inString {
			continue
		}

		switch r {
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '(':
			parenCount++
		case ')':
			parenCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}
	}

	return braceCount == 0 && parenCount == 0 && bracketCount == 0
}

// NEW: Basic tag balance check for HTML/XML
func (c *Client) hasBalancedTags(content string) bool {
	// This is a simplified check - a full parser would be better
	openTags := strings.Count(content, "<")
	closeTags := strings.Count(content, ">")
	return openTags == closeTags
}

// NEW: Basic Python structure validation
func (c *Client) hasValidPythonStructure(content string) bool {
	// Check for common Python patterns and that it doesn't end abruptly
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return false
	}

	lastLine := strings.TrimSpace(lines[len(lines)-1])

	// Python shouldn't end with these characters typically
	abruptEndings := []string{":", "\\", ",", "(", "[", "{"}
	for _, ending := range abruptEndings {
		if strings.HasSuffix(lastLine, ending) {
			return false
		}
	}

	return true
}

// NEW: Check for abrupt endings
func (c *Client) hasAbruptEnding(content string) bool {
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return true
	}

	lastChar := content[len(content)-1]
	abruptChars := ",(+*/-=&|<"

	return strings.ContainsRune(abruptChars, rune(lastChar))
}

// NEW: Check if response was complete based on finish reason
func (c *Client) wasResponseComplete(finishReason string) bool {
	incompleteReasons := []string{"length", "max_tokens", "max_output_tokens"}
	for _, reason := range incompleteReasons {
		if finishReason == reason {
			return false
		}
	}
	return true
}

func min(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
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
	GetFinishReason() string // ADD THIS
	IsComplete() bool        // ADD THIS
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
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"` // ADD THIS
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
	Content    []AnthropicContent `json:"content"`
	Usage      AnthropicUsage     `json:"usage"`
	StopReason string             `json:"stop_reason"` // ADD THIS
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

func (r *OpenAIResponse) GetFinishReason() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].FinishReason
	}
	return ""
}

func (r *OpenAIResponse) IsComplete() bool {
	reason := r.GetFinishReason()
	return reason == "stop" || reason == "end_turn" || reason == ""
}

func (r *AnthropicResponse) GetFinishReason() string {
	return r.StopReason
}

func (r *AnthropicResponse) IsComplete() bool {
	reason := r.GetFinishReason()
	return reason == "end_turn" || reason == "stop_sequence" || reason == ""
}
