package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

	// Convert to standard response
	return &types.AIResponse{
		Content:    apiResp.GetContent(),
		TokensUsed: apiResp.GetTokensUsed(),
		Model:      c.config.Model,
	}, nil
}

// buildPrompt constructs the full prompt with context
func (c *Client) buildPrompt(req types.AIRequest, contextFiles []*types.ContextFile) string {
	var prompt bytes.Buffer

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

	return prompt.String()
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
