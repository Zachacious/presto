package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader handles loading prompts from files
type Loader struct{}

// New creates a new prompt loader
func New() *Loader {
	return &Loader{}
}

// LoadPrompt loads a prompt from a file with variable substitution
func (l *Loader) LoadPrompt(promptPath string, variables map[string]string) (string, error) {
	content, err := os.ReadFile(promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", promptPath, err)
	}

	prompt := string(content)

	// Substitute variables in the format {{VARIABLE_NAME}}
	if variables != nil {
		for key, value := range variables {
			placeholder := fmt.Sprintf("{{%s}}", key)
			prompt = strings.ReplaceAll(prompt, placeholder, value)
		}
	}

	return strings.TrimSpace(prompt), nil
}

// ValidatePromptFile checks if a prompt file exists and is readable
func (l *Loader) ValidatePromptFile(path string) error {
	if path == "" {
		return fmt.Errorf("prompt file path is empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("prompt file not found: %s", path)
	}

	if info.IsDir() {
		return fmt.Errorf("prompt file path is a directory: %s", path)
	}

	// Check if file has reasonable size (not empty, not too large)
	if info.Size() == 0 {
		return fmt.Errorf("prompt file is empty: %s", path)
	}

	if info.Size() > 100*1024 { // 100KB limit for prompt files
		return fmt.Errorf("prompt file too large (>100KB): %s", path)
	}

	return nil
}

// GetBuiltinPrompts returns a list of built-in prompt templates
func (l *Loader) GetBuiltinPrompts() map[string]string {
	return map[string]string{
		"add-docs": `Add comprehensive documentation and comments to this code.
For functions, add:
- Clear description of what the function does
- Parameter descriptions
- Return value descriptions  
- Usage examples where helpful
- Any important notes about behavior or limitations

Keep the existing functionality unchanged.`,

		"add-error-handling": `Add robust error handling to this code:
- Add proper error checks for all operations that can fail
- Return meaningful error messages
- Use appropriate error types for the language
- Add logging where appropriate
- Handle edge cases and invalid inputs
- Follow best practices for the specific programming language

Keep existing functionality but make it production-ready.`,

		"optimize": `Optimize this code for better performance and readability:
- Improve algorithmic efficiency where possible
- Remove redundant code and operations
- Optimize data structures and memory usage
- Improve code organization and structure
- Add comments explaining optimizations
- Ensure the code remains readable and maintainable

Maintain the same functionality while improving performance.`,

		"modernize": `Modernize this code to use current best practices and language features:
- Update to modern syntax and idioms
- Use current standard library features
- Apply current design patterns
- Improve code organization
- Update naming conventions
- Remove deprecated features
- Add type hints/annotations where applicable

Keep the same functionality but make it current and idiomatic.`,

		"refactor": `Refactor this code to improve maintainability and follow best practices:
- Extract reusable functions/methods
- Improve naming for clarity
- Reduce code duplication
- Improve separation of concerns
- Add appropriate abstractions
- Simplify complex logic
- Follow language-specific conventions

Keep functionality identical while improving code quality.`,

		"add-tests": `Add comprehensive unit tests for this code:
- Test all public functions/methods
- Include edge cases and error conditions
- Add positive and negative test cases
- Use appropriate testing framework for the language
- Include setup and teardown as needed
- Add descriptive test names and comments
- Aim for high code coverage

Generate complete, runnable tests.`,

		"convert-language": `Convert this code to {{TARGET_LANGUAGE}}:
- Maintain exact same functionality
- Use idiomatic {{TARGET_LANGUAGE}} patterns
- Include appropriate imports/dependencies
- Follow {{TARGET_LANGUAGE}} naming conventions
- Use {{TARGET_LANGUAGE}} standard library equivalents
- Add comments explaining any significant changes
- Ensure the code compiles and runs correctly

Provide a complete, working translation.`,
	}
}

// SaveBuiltinPrompt saves a built-in prompt template to file
func (l *Loader) SaveBuiltinPrompt(name, outputPath string) error {
	prompts := l.GetBuiltinPrompts()

	prompt, exists := prompts[name]
	if !exists {
		return fmt.Errorf("unknown built-in prompt: %s", name)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	err := os.WriteFile(outputPath, []byte(prompt), 0644)
	if err != nil {
		return fmt.Errorf("failed to write prompt file: %w", err)
	}

	return nil
}

// ListTemplates returns the names of all built-in templates
func (l *Loader) ListTemplates() []string {
	prompts := l.GetBuiltinPrompts()
	var names []string
	for name := range prompts {
		names = append(names, name)
	}
	return names
}

// GetTemplate returns a specific built-in template
func (l *Loader) GetTemplate(name string) (string, error) {
	prompts := l.GetBuiltinPrompts()
	template, exists := prompts[name]
	if !exists {
		return "", fmt.Errorf("template '%s' not found", name)
	}
	return template, nil
}
