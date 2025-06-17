package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Zachacious/presto/internal/config"
	"github.com/Zachacious/presto/pkg/types"
)

// Client handles AI requests via OpenRouter
type Client struct {
	config     *config.AIConfig
	httpClient *http.Client
}

// OpenRouterRequest represents the request format for OpenRouter API
type OpenRouterRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenRouterResponse represents the response from OpenRouter API
type OpenRouterResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// New creates a new AI client
func New(cfg *config.AIConfig) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// ProcessContent sends content to AI for processing
func (c *Client) ProcessContent(req types.AIRequest, contextFiles []*types.ContextFile) (*types.AIResponse, error) {
	startTime := time.Now()

	prompt := c.buildPrompt(req, contextFiles)

	openRouterReq := OpenRouterRequest{
		Model: c.config.Model,
		Messages: []Message{
			{
				Role:    "system",
				Content: c.getSystemPrompt(req.Mode),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	response, err := c.makeRequest(openRouterReq)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, fmt.Errorf("API error: %s", response.Error.Message)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no response choices returned")
	}

	return &types.AIResponse{
		Content:    response.Choices[0].Message.Content,
		TokensUsed: response.Usage.TotalTokens,
		Model:      response.Model,
		Duration:   time.Since(startTime),
	}, nil
}

// getSystemPrompt returns the appropriate system prompt based on mode
func (c *Client) getSystemPrompt(mode types.ProcessingMode) string {
	switch mode {
	case types.ModeGenerate:
		return `You are a helpful assistant that generates new files based on context and prompts. 
Study the provided context files to understand patterns, architecture, and coding style. 
Generate complete, working code that follows the same patterns and conventions.
Return only the generated code unless specifically asked to include explanations.`

	case types.ModeTransform:
		return `You are a helpful assistant that transforms files according to specific instructions. 
Follow the user's prompt exactly and return only the processed content unless specifically asked to include explanations.
When context files are provided, use them to understand coding patterns and maintain consistency.`

	default:
		return `You are a helpful assistant that processes files according to specific instructions.
Follow the user's prompt exactly and maintain the same functionality unless asked to modify it.`
	}
}

// buildPrompt constructs the full prompt for the AI
func (c *Client) buildPrompt(req types.AIRequest, contextFiles []*types.ContextFile) string {
	var prompt bytes.Buffer

	// Add context files if provided
	if len(contextFiles) > 0 {
		prompt.WriteString("CONTEXT FILES:\n")
		prompt.WriteString("The following files provide context about the codebase, patterns, and requirements:\n\n")

		for i, ctx := range contextFiles {
			prompt.WriteString(fmt.Sprintf("[Context %d: %s (%s)]\n", i+1, ctx.Label, ctx.Language))
			prompt.WriteString("```\n")
			prompt.WriteString(ctx.Content)
			prompt.WriteString("\n```\n\n")
		}

		prompt.WriteString("END CONTEXT\n\n")
	}

	// Add the main prompt
	prompt.WriteString("TASK:\n")
	prompt.WriteString(req.Prompt)
	prompt.WriteString("\n\n")

	// Add target content if in transform mode
	if req.Mode == types.ModeTransform && req.Content != "" {
		if req.Language != types.LangUnknown {
			prompt.WriteString(fmt.Sprintf("TARGET FILE (%s):\n", req.Language))
		} else {
			prompt.WriteString("TARGET FILE:\n")
		}
		prompt.WriteString("```\n")
		prompt.WriteString(req.Content)
		prompt.WriteString("\n```\n\n")
	}

	// Add mode-specific instructions
	switch req.Mode {
	case types.ModeGenerate:
		prompt.WriteString("Please generate the new file content based on the context and requirements above.\n")
	case types.ModeTransform:
		prompt.WriteString("Please process the target file according to the task above.\n")
	}

	if len(contextFiles) > 0 {
		prompt.WriteString("Use the context files to understand patterns, style, and architecture.\n")
	}

	return prompt.String()
}

// makeRequest makes an HTTP request to the OpenRouter API
func (c *Client) makeRequest(req OpenRouterRequest) (*OpenRouterResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.GetAPIKey())
	httpReq.Header.Set("HTTP-Referer", "https://github.com/yourusername/presto")
	httpReq.Header.Set("X-Title", "Presto - AI File Processor")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response OpenRouterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// ValidateConfig checks if the AI configuration is valid
func (c *Client) ValidateConfig() error {
	if c.config.GetAPIKey() == "" {
		return fmt.Errorf("API key not found in environment variable %s", c.config.APIKeyEnv)
	}

	if c.config.Model == "" {
		return fmt.Errorf("model not specified")
	}

	return nil
}
