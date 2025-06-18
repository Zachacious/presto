package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zachacious/presto/pkg/types"
	"gopkg.in/yaml.v3"
)

// Manager handles command operations
type Manager struct {
	commands    map[string]*types.Command
	commandsDir string
}

// New creates a new command manager
func New() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	commandsDir := filepath.Join(homeDir, ".presto", "commands")
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create commands directory: %w", err)
	}

	m := &Manager{
		commands:    make(map[string]*types.Command),
		commandsDir: commandsDir,
	}

	// Load built-in commands
	if err := m.loadBuiltinCommands(); err != nil {
		return nil, fmt.Errorf("failed to load built-in commands: %w", err)
	}

	// Load user commands
	if err := m.loadUserCommands(); err != nil {
		return nil, fmt.Errorf("failed to load user commands: %w", err)
	}

	return m, nil
}

// GetCommand retrieves a command by name
func (m *Manager) GetCommand(name string) (*types.Command, error) {
	cmd, exists := m.commands[name]
	if !exists {
		return nil, fmt.Errorf("command '%s' not found", name)
	}
	return m.copyCommand(cmd), nil
}

// GetBuiltinCommands returns all built-in commands
func (m *Manager) GetBuiltinCommands() map[string]*types.Command {
	builtins := make(map[string]*types.Command)
	for name, cmd := range m.commands {
		if m.IsBuiltin(name) {
			builtins[name] = m.copyCommand(cmd)
		}
	}
	return builtins
}

// GetUserCommands returns all user commands
func (m *Manager) GetUserCommands() map[string]*types.Command {
	userCmds := make(map[string]*types.Command)
	for name, cmd := range m.commands {
		if !m.IsBuiltin(name) {
			userCmds[name] = m.copyCommand(cmd)
		}
	}
	return userCmds
}

// IsBuiltin checks if a command is built-in
func (m *Manager) IsBuiltin(name string) bool {
	builtinNames := []string{"add-docs", "add-logging", "optimize", "modernize", "cleanup", "explain", "summarize", "convert"}
	for _, builtin := range builtinNames {
		if name == builtin {
			return true
		}
	}
	return false
}

// SaveCommand saves a user command
func (m *Manager) SaveCommand(cmd *types.Command) error {
	if m.IsBuiltin(cmd.Name) {
		return fmt.Errorf("cannot override built-in command: %s", cmd.Name)
	}

	// Save to file
	filePath := filepath.Join(m.commandsDir, cmd.Name+".yaml")
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

// DeleteCommand deletes a user command
func (m *Manager) DeleteCommand(name string) error {
	if m.IsBuiltin(name) {
		return fmt.Errorf("cannot delete built-in command: %s", name)
	}

	filePath := filepath.Join(m.commandsDir, name+".yaml")
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
	if cmd.Mode != "" {
		opts.Mode = cmd.Mode
	}
	if cmd.Prompt != "" {
		opts.AIPrompt = cmd.Prompt
	}
	if cmd.PromptFile != "" {
		opts.PromptFile = cmd.PromptFile
	}

	// Apply command options
	if cmd.Options.OutputMode != "" {
		opts.OutputMode = types.OutputMode(cmd.Options.OutputMode)
	}
	if cmd.Options.OutputSuffix != "" {
		opts.OutputSuffix = cmd.Options.OutputSuffix
	}
	if cmd.Options.FilePattern != "" {
		opts.FilePattern = cmd.Options.FilePattern
	}
	if cmd.Options.ExcludePattern != "" {
		opts.ExcludePattern = cmd.Options.ExcludePattern
	}
	if len(cmd.Options.ContextPatterns) > 0 {
		opts.ContextPatterns = append(opts.ContextPatterns, cmd.Options.ContextPatterns...)
	}
	if len(cmd.Options.ContextFiles) > 0 {
		opts.ContextFiles = append(opts.ContextFiles, cmd.Options.ContextFiles...)
	}
	if cmd.Options.Recursive {
		opts.Recursive = cmd.Options.Recursive
	}
	if cmd.Options.RemoveComments {
		opts.RemoveComments = cmd.Options.RemoveComments
	}
	if cmd.Options.BackupOriginal {
		opts.BackupOriginal = cmd.Options.BackupOriginal
	}
	if cmd.Options.Model != "" {
		opts.Model = cmd.Options.Model
	}
	if cmd.Options.Temperature != 0 {
		opts.Temperature = cmd.Options.Temperature
	}
	if cmd.Options.MaxTokens != 0 {
		opts.MaxTokens = cmd.Options.MaxTokens
	}

	return nil
}

// SubstituteVariables substitutes variables in a command
func (m *Manager) SubstituteVariables(cmd *types.Command, variables map[string]string) {
	// Merge command variables with provided variables
	allVars := make(map[string]string)
	for k, v := range cmd.Variables {
		allVars[k] = v
	}
	for k, v := range variables {
		allVars[k] = v
	}

	// Substitute in prompt
	cmd.Prompt = m.substituteString(cmd.Prompt, allVars)
	cmd.PromptFile = m.substituteString(cmd.PromptFile, allVars)
}

// GenerateCommandTemplate generates a command template
func (m *Manager) GenerateCommandTemplate(name string) *types.Command {
	return &types.Command{
		Name:        name,
		Description: fmt.Sprintf("Custom command: %s", name),
		Mode:        types.ModeTransform,
		Prompt:      "Your prompt here",
		Options: types.CommandOptions{
			OutputMode: "separate",
			Recursive:  true,
		},
		Variables: make(map[string]string),
	}
}

// copyCommand creates a deep copy of a command
func (m *Manager) copyCommand(cmd *types.Command) *types.Command {
	cmdCopy := *cmd

	// Copy slices
	cmdCopy.Options.ContextPatterns = make([]string, len(cmd.Options.ContextPatterns))
	copy(cmdCopy.Options.ContextPatterns, cmd.Options.ContextPatterns)

	cmdCopy.Options.ContextFiles = make([]string, len(cmd.Options.ContextFiles))
	copy(cmdCopy.Options.ContextFiles, cmd.Options.ContextFiles)

	// Copy variables map
	cmdCopy.Variables = make(map[string]string)
	for k, v := range cmd.Variables {
		cmdCopy.Variables[k] = v
	}

	return &cmdCopy
}

// substituteString substitutes variables in a string
func (m *Manager) substituteString(s string, variables map[string]string) string {
	result := s
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
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

// loadUserCommands loads user commands from files
func (m *Manager) loadUserCommands() error {
	entries, err := os.ReadDir(m.commandsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		filePath := filepath.Join(m.commandsDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var cmd types.Command
		if err := yaml.Unmarshal(data, &cmd); err != nil {
			continue
		}

		if cmd.Name != "" {
			m.commands[cmd.Name] = &cmd
		}
	}

	return nil
}
