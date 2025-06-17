package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zachacious/presto/pkg/types"
	"gopkg.in/yaml.v3"
)

// Manager handles prefab commands
type Manager struct {
	commands map[string]*types.Command
	userDir  string
}

// New creates a new command manager
func New() (*Manager, error) {
	userDir, err := getUserCommandsDir()
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		commands: make(map[string]*types.Command),
		userDir:  userDir,
	}

	// Load built-in commands
	if err := manager.loadBuiltinCommands(); err != nil {
		return nil, err
	}

	// Load user commands
	if err := manager.loadUserCommands(); err != nil {
		return nil, err
	}

	return manager, nil
}

// GetCommand retrieves a command by name
func (m *Manager) GetCommand(name string) (*types.Command, error) {
	cmd, exists := m.commands[name]
	if !exists {
		return nil, fmt.Errorf("command '%s' not found", name)
	}

	// Create a copy to avoid modifying the original
	cmdCopy := *cmd

	// Deep copy slices and maps
	if cmd.Options.ContextPatterns != nil {
		cmdCopy.Options.ContextPatterns = make([]string, len(cmd.Options.ContextPatterns))
		copy(cmdCopy.Options.ContextPatterns, cmd.Options.ContextPatterns)
	}

	if cmd.Options.ContextFiles != nil {
		cmdCopy.Options.ContextFiles = make([]string, len(cmd.Options.ContextFiles))
		copy(cmdCopy.Options.ContextFiles, cmd.Options.ContextFiles)
	}

	if cmd.Variables != nil {
		cmdCopy.Variables = make(map[string]string)
		for k, v := range cmd.Variables {
			cmdCopy.Variables[k] = v
		}
	}

	return &cmdCopy, nil
}

// ListCommands returns all available commands
func (m *Manager) ListCommands() map[string]*types.Command {
	result := make(map[string]*types.Command)
	for k, v := range m.commands {
		result[k] = v
	}
	return result
}

// SaveCommand saves a user command
func (m *Manager) SaveCommand(cmd *types.Command) error {
	if err := m.ensureUserDir(); err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.yaml", cmd.Name)
	filePath := filepath.Join(m.userDir, filename)

	data, err := yaml.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write command file: %w", err)
	}

	// Add to memory
	m.commands[cmd.Name] = cmd

	return nil
}

// DeleteCommand removes a user command
func (m *Manager) DeleteCommand(name string) error {
	filename := fmt.Sprintf("%s.yaml", name)
	filePath := filepath.Join(m.userDir, filename)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete command file: %w", err)
	}

	delete(m.commands, name)
	return nil
}

// ApplyCommand applies a command to processing options
func (m *Manager) ApplyCommand(name string, opts *types.ProcessingOptions) error {
	cmd, err := m.GetCommand(name)
	if err != nil {
		return err
	}

	// Apply command settings to options
	opts.Mode = cmd.Mode

	// Only override if not already set
	if opts.AIPrompt == "" && cmd.Prompt != "" {
		opts.AIPrompt = cmd.Prompt
	}
	if opts.PromptFile == "" && cmd.PromptFile != "" {
		opts.PromptFile = cmd.PromptFile
	}

	// Apply command options (only if not explicitly set by user)
	if cmd.Options.OutputMode != "" {
		opts.OutputMode = types.OutputMode(cmd.Options.OutputMode)
	}
	if cmd.Options.OutputSuffix != "" && opts.OutputSuffix == ".presto" {
		opts.OutputSuffix = cmd.Options.OutputSuffix
	}
	if cmd.Options.FilePattern != "" && opts.FilePattern == "" {
		opts.FilePattern = cmd.Options.FilePattern
	}
	if cmd.Options.ExcludePattern != "" && opts.ExcludePattern == "" {
		opts.ExcludePattern = cmd.Options.ExcludePattern
	}
	if len(cmd.Options.ContextPatterns) > 0 && len(opts.ContextPatterns) == 0 {
		opts.ContextPatterns = append(opts.ContextPatterns, cmd.Options.ContextPatterns...)
	}
	if len(cmd.Options.ContextFiles) > 0 && len(opts.ContextFiles) == 0 {
		opts.ContextFiles = append(opts.ContextFiles, cmd.Options.ContextFiles...)
	}
	if cmd.Options.Model != "" && opts.Model == "" {
		opts.Model = cmd.Options.Model
	}
	if cmd.Options.Temperature != 0 && opts.Temperature == 0 {
		opts.Temperature = cmd.Options.Temperature
	}
	if cmd.Options.MaxTokens != 0 && opts.MaxTokens == 0 {
		opts.MaxTokens = cmd.Options.MaxTokens
	}

	// Boolean options from command (can be overridden by CLI)
	if cmd.Options.Recursive {
		opts.Recursive = true
	}
	if cmd.Options.RemoveComments {
		opts.RemoveComments = true
	}
	if cmd.Options.BackupOriginal {
		opts.BackupOriginal = true
	}

	return nil
}

// SubstituteVariables replaces variables in the command
func (m *Manager) SubstituteVariables(cmd *types.Command, variables map[string]string) {
	// Merge command variables with provided variables (provided variables take precedence)
	allVars := make(map[string]string)
	if cmd.Variables != nil {
		for k, v := range cmd.Variables {
			allVars[k] = v
		}
	}
	if variables != nil {
		for k, v := range variables {
			allVars[k] = v
		}
	}

	// Substitute in prompt
	cmd.Prompt = substituteString(cmd.Prompt, allVars)
	cmd.PromptFile = substituteString(cmd.PromptFile, allVars)

	// Substitute in options
	cmd.Options.OutputSuffix = substituteString(cmd.Options.OutputSuffix, allVars)
	cmd.Options.FilePattern = substituteString(cmd.Options.FilePattern, allVars)
	cmd.Options.ExcludePattern = substituteString(cmd.Options.ExcludePattern, allVars)

	for i, pattern := range cmd.Options.ContextPatterns {
		cmd.Options.ContextPatterns[i] = substituteString(pattern, allVars)
	}
	for i, file := range cmd.Options.ContextFiles {
		cmd.Options.ContextFiles[i] = substituteString(file, allVars)
	}
}

// loadBuiltinCommands loads built-in commands
func (m *Manager) loadBuiltinCommands() error {
	builtins := []*types.Command{
		{
			Name:        "add-docs",
			Description: "Add comprehensive documentation and comments",
			Mode:        types.ModeTransform,
			Prompt:      "Add comprehensive documentation and comments to this content. For functions and methods, add clear descriptions of what they do, their parameters, return values, and usage examples where helpful. Keep the existing functionality unchanged.",
			Options: types.CommandOptions{
				OutputMode:   "separate",
				OutputSuffix: ".documented",
				Recursive:    true,
			},
		},
		{
			Name:        "add-logging",
			Description: "Add logging statements throughout the content",
			Mode:        types.ModeTransform,
			Prompt:      "Add appropriate logging statements throughout this content. Include logging for important operations, errors, and key decision points. Use appropriate log levels and ensure sensitive information is not logged.",
			Options: types.CommandOptions{
				OutputMode:   "separate",
				OutputSuffix: ".with-logging",
				Recursive:    true,
			},
		},
		{
			Name:        "optimize",
			Description: "Optimize content for better performance and readability",
			Mode:        types.ModeTransform,
			Prompt:      "Optimize this content for better performance and readability. Improve efficiency where possible, remove redundancy, and enhance organization while maintaining the same functionality.",
			Options: types.CommandOptions{
				OutputMode:   "separate",
				OutputSuffix: ".optimized",
				Recursive:    true,
			},
		},
		{
			Name:        "modernize",
			Description: "Update content to use current best practices",
			Mode:        types.ModeTransform,
			Prompt:      "Modernize this content to use current best practices and conventions. Update syntax, patterns, and approaches to follow contemporary standards while maintaining the same functionality.",
			Options: types.CommandOptions{
				OutputMode:   "separate",
				OutputSuffix: ".modern",
				Recursive:    true,
			},
		},
		{
			Name:        "cleanup",
			Description: "Clean and format content following best practices",
			Mode:        types.ModeTransform,
			Prompt:      "Clean up this content by improving formatting, removing dead or redundant sections, standardizing style, and ensuring it follows best practices. Maintain all functionality.",
			Options: types.CommandOptions{
				OutputMode:   "separate",
				OutputSuffix: ".clean",
				Recursive:    true,
			},
		},
		{
			Name:        "explain",
			Description: "Add explanations and clarifying comments",
			Mode:        types.ModeTransform,
			Prompt:      "Add clear explanations and clarifying comments throughout this content. Explain complex logic, add context for decisions, and make the content more understandable for others.",
			Options: types.CommandOptions{
				OutputMode:   "separate",
				OutputSuffix: ".explained",
				Recursive:    true,
			},
		},
		{
			Name:        "summarize",
			Description: "Create a summary of the content",
			Mode:        types.ModeGenerate,
			Prompt:      "Create a comprehensive summary of the content in the context files. Include key points, main concepts, and important details in a well-organized format.",
			Options: types.CommandOptions{
				OutputMode: "file",
			},
		},
		{
			Name:        "convert",
			Description: "Convert content to a different format or style",
			Mode:        types.ModeTransform,
			Prompt:      "Convert this content to {{TARGET_FORMAT}}. Maintain the same information and functionality while adapting to the new format or style requirements.",
			Options: types.CommandOptions{
				OutputMode:   "separate",
				OutputSuffix: ".converted",
			},
			Variables: map[string]string{
				"TARGET_FORMAT": "markdown",
			},
		},
	}

	for _, cmd := range builtins {
		m.commands[cmd.Name] = cmd
	}

	return nil
}

// loadUserCommands loads user-defined commands
func (m *Manager) loadUserCommands() error {
	if _, err := os.Stat(m.userDir); os.IsNotExist(err) {
		return nil // No user commands directory yet
	}

	files, err := os.ReadDir(m.userDir)
	if err != nil {
		return fmt.Errorf("failed to read user commands directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".yaml") && !strings.HasSuffix(file.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(m.userDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip problematic files
		}

		var cmd types.Command
		if err := yaml.Unmarshal(data, &cmd); err != nil {
			continue // Skip malformed files
		}

		if cmd.Name != "" {
			m.commands[cmd.Name] = &cmd
		}
	}

	return nil
}

// getUserCommandsDir returns the user commands directory
func getUserCommandsDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".presto", "commands"), nil
}

// ensureUserDir creates the user commands directory if it doesn't exist
func (m *Manager) ensureUserDir() error {
	return os.MkdirAll(m.userDir, 0755)
}

// substituteString replaces {{VAR}} patterns with values
func substituteString(s string, vars map[string]string) string {
	result := s
	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// GenerateCommandTemplate generates a template for creating new commands
func (m *Manager) GenerateCommandTemplate(name string) *types.Command {
	return &types.Command{
		Name:        name,
		Description: "Description of what this command does",
		Mode:        types.ModeTransform,
		Prompt:      "Your AI prompt here. Use {{VARIABLES}} for substitution.",
		Options: types.CommandOptions{
			OutputMode:   "separate",
			OutputSuffix: ".processed",
			Recursive:    true,
		},
		Variables: map[string]string{
			"EXAMPLE_VAR": "default_value",
		},
	}
}

// IsBuiltin checks if a command is a built-in command
func (m *Manager) IsBuiltin(name string) bool {
	builtinNames := []string{
		"add-docs", "add-errors", "optimize", "js2ts", "ts2js",
		"modernize", "create-service", "create-test", "clean-code", "add-logging",
	}

	for _, builtin := range builtinNames {
		if name == builtin {
			return true
		}
	}
	return false
}

// GetUserCommands returns only user-defined commands
func (m *Manager) GetUserCommands() map[string]*types.Command {
	userCommands := make(map[string]*types.Command)
	for name, cmd := range m.commands {
		if !m.IsBuiltin(name) {
			userCommands[name] = cmd
		}
	}
	return userCommands
}

// GetBuiltinCommands returns only built-in commands
func (m *Manager) GetBuiltinCommands() map[string]*types.Command {
	builtinCommands := make(map[string]*types.Command)
	for name, cmd := range m.commands {
		if m.IsBuiltin(name) {
			builtinCommands[name] = cmd
		}
	}
	return builtinCommands
}
