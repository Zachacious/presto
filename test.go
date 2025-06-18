// Package commands provides a manager for handling and applying various types of commands
// within the Presto application, including built-in and user-defined commands.
package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zachacious/presto/pkg/types"
	"gopkg.in/yaml.v3"
)

// Manager handles the loading, storage, retrieval, and application of commands.
// It manages both built-in commands and user-defined commands stored on the filesystem.
type Manager struct {
	commands    map[string]*types.Command // commands stores all loaded commands, keyed by their name.
	commandsDir string                    // commandsDir is the directory where user-defined commands are stored.
}

// New creates and initializes a new Manager instance.
// It sets up the directory for user commands and loads both built-in and existing user commands.
//
// Returns:
//   - *Manager: A pointer to the initialized Manager.
//   - error: An error if the commands directory cannot be created or if commands fail to load.
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

// GetCommand retrieves a command by its name.
// It returns a deep copy of the command to prevent external modifications to the stored command.
//
// Parameters:
//   - name: The name of the command to retrieve.
//
// Returns:
//   - *types.Command: A deep copy of the requested command.
//   - error: An error if the command with the given name is not found.
func (m *Manager) GetCommand(name string) (*types.Command, error) {
	cmd, exists := m.commands[name]
	if !exists {
		return nil, fmt.Errorf("command '%s' not found", name)
	}
	return m.copyCommand(cmd), nil
}

// GetBuiltinCommands returns a map of all built-in commands.
// Each command in the returned map is a deep copy of the original.
//
// Returns:
//   - map[string]*types.Command: A map where keys are command names and values are deep copies of built-in commands.
func (m *Manager) GetBuiltinCommands() map[string]*types.Command {
	builtins := make(map[string]*types.Command)
	for name, cmd := range m.commands {
		if m.IsBuiltin(name) {
			builtins[name] = m.copyCommand(cmd)
		}
	}
	return builtins
}

// GetUserCommands returns a map of all user-defined commands.
// Each command in the returned map is a deep copy of the original.
//
// Returns:
//   - map[string]*types.Command: A map where keys are command names and values are deep copies of user commands.
func (m *Manager) GetUserCommands() map[string]*types.Command {
	userCmds := make(map[string]*types.Command)
	for name, cmd := range m.commands {
		if !m.IsBuiltin(name) {
			userCmds[name] = m.copyCommand(cmd)
		}
	}
	return userCmds
}

// IsBuiltin checks if a command with the given name is a built-in command.
//
// Parameters:
//   - name: The name of the command to check.
//
// Returns:
//   - bool: True if the command is built-in, false otherwise.
func (m *Manager) IsBuiltin(name string) bool {
	builtinNames := []string{"add-docs", "add-logging", "optimize", "modernize", "cleanup", "explain", "summarize", "convert"}
	for _, builtin := range builtinNames {
		if name == builtin {
			return true
		}
	}
	return false
}

// SaveCommand saves a user-defined command to the filesystem and adds it to the in-memory command store.
// Built-in commands cannot be overwritten.
//
// Parameters:
//   - cmd: The types.Command object to save.
//
// Returns:
//   - error: An error if the command is built-in, or if marshaling/writing the file fails.
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

// DeleteCommand deletes a user-defined command from the filesystem and removes it from the in-memory store.
// Built-in commands cannot be deleted.
//
// Parameters:
//   - name: The name of the command to delete.
//
// Returns:
//   - error: An error if the command is built-in, or if deleting the file fails for reasons other than it not existing.
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

// ApplyCommand applies the settings of a specified command to a ProcessingOptions struct.
// This function updates the fields of `opts` based on the command's configuration.
//
// Parameters:
//   - name: The name of the command to apply.
//   - opts: A pointer to the types.ProcessingOptions struct to which the command settings will be applied.
//
// Returns:
//   - error: An error if the command is not found.
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

// SubstituteVariables replaces placeholders in a command's prompt and prompt file path
// with values from the provided variables map and the command's own variables.
// Command variables take precedence over provided variables if keys conflict.
// Placeholders are in the format `{{KEY}}`.
//
// Parameters:
//   - cmd: The types.Command object in which variables will be substituted.
//   - variables: A map of variable names to their values for substitution.
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

// GenerateCommandTemplate creates a new types.Command object with default values,
// suitable for use as a template for new user-defined commands.
//
// Parameters:
//   - name: The desired name for the new command.
//
// Returns:
//   - *types.Command: A pointer to the newly generated command template.
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

// copyCommand creates a deep copy of a types.Command object.
// This is used to ensure that modifications to retrieved commands do not affect
// the stored in-memory versions.
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

// substituteString replaces all occurrences of `{{key}}` placeholders in a string
// with their corresponding values from the provided variables map.
func (m *Manager) substituteString(s string, variables map[string]string) string {
	result := s
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// loadBuiltinCommands initializes the Manager's command map with predefined built-in commands.
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
				Recursive