package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Zachacious/presto/internal/commands"
	"github.com/Zachacious/presto/internal/config"
	"github.com/Zachacious/presto/internal/processor"
	"github.com/Zachacious/presto/pkg/types"
)

const (
	version = "0.1.0"
)

func main() {
	// Command-line flags
	var (
		showVersion   = flag.Bool("version", false, "Show version information")
		showHelp      = flag.Bool("help", false, "Show help information")
		listCommands  = flag.Bool("list-commands", false, "List all available commands")
		showCommand   = flag.String("show-command", "", "Show details for a specific command")
		deleteCommand = flag.String("delete-command", "", "Delete a user command")
		editCommand   = flag.String("edit-command", "", "Edit a user command")

		// Processing options
		promptText     = flag.String("prompt", "", "AI prompt text")
		promptFile     = flag.String("prompt-file", "", "File containing AI prompt")
		commandName    = flag.String("cmd", "", "Use a predefined command")
		inputPath      = flag.String("input", "", "Input file or directory path")
		outputPath     = flag.String("output-file", "", "Output file path (for generate mode)")
		outputMode     = flag.String("output", "separate", "Output mode: inplace|separate|stdout|file")
		outputSuffix   = flag.String("suffix", ".presto", "Suffix for output files in separate mode")
		recursive      = flag.Bool("recursive", false, "Process directories recursively")
		filePattern    = flag.String("pattern", "", "File pattern regex to match")
		excludePattern = flag.String("exclude", "", "File pattern regex to exclude")
		generateMode   = flag.Bool("generate", false, "Generate new content instead of transforming")
		removeComments = flag.Bool("remove-comments", false, "Remove comments from input before processing")

		// Context options
		contextFiles    = flag.String("context", "", "Comma-separated context file paths")
		contextPatterns = flag.String("context-pattern", "", "Comma-separated context file patterns")

		// AI options
		model       = flag.String("model", "", "AI model to use")
		temperature = flag.Float64("temperature", 0, "AI temperature (0.0-2.0)")
		maxTokens   = flag.Int("max-tokens", 0, "Maximum tokens for AI response")

		// Processing options
		dryRun         = flag.Bool("dry-run", false, "Show what would be done without making changes")
		verbose        = flag.Bool("verbose", false, "Verbose output")
		maxConcurrent  = flag.Int("concurrent", 3, "Maximum concurrent file processing")
		backupOriginal = flag.Bool("backup", false, "Create backup of original files")
		saveCommandAs  = flag.String("save-command", "", "Save current options as a named command")

		// Variable substitution
		variables = flag.String("var", "", "Variables for command substitution (VAR=value,VAR2=value2)")
	)

	// Handle configure command first (before flag parsing)
	if len(os.Args) > 1 && os.Args[1] == "configure" {
		if err := config.ConfigureInteractive(); err != nil {
			log.Fatalf("❌ Configuration failed: %v", err)
		}
		return
	}

	flag.Parse()

	// Handle utility commands first
	if *showVersion {
		fmt.Printf("Presto v%s\n", version)
		return
	}

	if *showHelp {
		showHelpText()
		return
	}

	// Initialize command manager
	cmdManager, err := commands.New()
	if err != nil {
		log.Fatalf("❌ Failed to initialize command manager: %v", err)
	}

	if *listCommands {
		handleListCommands(cmdManager)
		return
	}

	if *showCommand != "" {
		handleShowCommand(cmdManager, *showCommand)
		return
	}

	if *deleteCommand != "" {
		handleDeleteCommand(cmdManager, *deleteCommand)
		return
	}

	if *editCommand != "" {
		handleEditCommand(cmdManager, *editCommand)
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("❌ Configuration error: %v\n\n", err)
		fmt.Println("To fix this, you can:")
		fmt.Println("1. Run: presto configure")
		fmt.Println("2. Set environment variable: export OPENAI_API_KEY=\"your-key\"")
		fmt.Println("3. Edit config file: ~/.presto/config.yaml")
		os.Exit(1)
	}

	// Parse context files and patterns
	var contextFileList []string
	var contextPatternList []string

	if *contextFiles != "" {
		contextFileList = strings.Split(*contextFiles, ",")
		for i, file := range contextFileList {
			contextFileList[i] = strings.TrimSpace(file)
		}
	}

	if *contextPatterns != "" {
		contextPatternList = strings.Split(*contextPatterns, ",")
		for i, pattern := range contextPatternList {
			contextPatternList[i] = strings.TrimSpace(pattern)
		}
	}

	// Parse variables
	varMap := make(map[string]string)
	if *variables != "" {
		pairs := strings.Split(*variables, ",")
		for _, pair := range pairs {
			parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(parts) == 2 {
				varMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Build processing options - use types.ProcessingOptions
	opts := &types.ProcessingOptions{
		Mode:            types.ModeTransform,
		AIPrompt:        *promptText,
		PromptFile:      *promptFile,
		InputPath:       *inputPath,
		OutputPath:      *outputPath,
		OutputMode:      types.OutputMode(*outputMode),
		OutputSuffix:    *outputSuffix,
		ContextFiles:    contextFileList,
		ContextPatterns: contextPatternList,
		Recursive:       *recursive,
		FilePattern:     *filePattern,
		ExcludePattern:  *excludePattern,
		RemoveComments:  *removeComments,
		DryRun:          *dryRun,
		Verbose:         *verbose,
		MaxConcurrent:   *maxConcurrent,
		BackupOriginal:  *backupOriginal,
		Model:           *model,
		Temperature:     *temperature,
		MaxTokens:       *maxTokens,
	}

	if *generateMode {
		opts.Mode = types.ModeGenerate
	}

	// Apply command if specified
	if *commandName != "" {
		if err := cmdManager.ApplyCommand(*commandName, opts); err != nil {
			log.Fatalf("❌ Failed to apply command '%s': %v", *commandName, err)
		}

		// Apply variable substitutions if command supports them
		if cmd, err := cmdManager.GetCommand(*commandName); err == nil && len(varMap) > 0 {
			cmdManager.SubstituteVariables(cmd, varMap)
			opts.AIPrompt = cmd.Prompt
			opts.PromptFile = cmd.PromptFile
		}
	}

	// Validate required options
	if opts.AIPrompt == "" && opts.PromptFile == "" {
		log.Fatal("❌ Either --prompt or --prompt-file is required")
	}

	if opts.InputPath == "" && opts.Mode == types.ModeTransform {
		log.Fatal("❌ --input is required for transform mode")
	}

	if opts.OutputPath == "" && opts.Mode == types.ModeGenerate {
		log.Fatal("❌ --output-file is required for generate mode")
	}

	// Handle save command option
	if *saveCommandAs != "" {
		handleSaveCommand(cmdManager, *saveCommandAs, opts)
		return
	}

	// Handle missing API key gracefully
	if cfg.AI.APIKey == "" {
		fmt.Println("⚠️  No OpenRouter API key found.")
		fmt.Println("You can:")
		fmt.Println("1. Set environment variable: export OPENROUTER_API_KEY=\"your-key\"")
		fmt.Println("2. Add to config file: ~/.presto/config.yaml")
		fmt.Println("3. Use --api-key flag")
		fmt.Println("4. Run 'presto setup' for interactive configuration")
		fmt.Println()
		fmt.Print("Enter API key now (or Ctrl+C to exit): ")

		var apiKey string
		if _, err := fmt.Scanln(&apiKey); err != nil {
			log.Fatal("❌ API key required")
		}

		cfg.AI.APIKey = strings.TrimSpace(apiKey)
		if cfg.AI.APIKey == "" {
			log.Fatal("❌ API key cannot be empty")
		}

		// Ask if they want to save it
		fmt.Print("💾 Save API key to config file? (y/N): ")
		var save string
		fmt.Scanln(&save)
		if strings.ToLower(save) == "y" || strings.ToLower(save) == "yes" {
			if err := config.SaveAPIKey(cfg.AI.APIKey); err != nil {
				fmt.Printf("⚠️  Failed to save API key: %v\n", err)
			} else {
				fmt.Println("✅ API key saved to ~/.presto/config.yaml")
			}
		}
	}

	// Apply config defaults to options that weren't explicitly set
	if opts.Model == "" {
		opts.Model = cfg.AI.Model
	}
	if opts.Temperature == 0 {
		opts.Temperature = cfg.AI.Temperature
	}
	if opts.MaxTokens == 0 {
		opts.MaxTokens = cfg.AI.MaxTokens
	}

	// Initialize processor
	proc, err := processor.New(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize processor: %v", err)
	}

	// Process files
	results, err := proc.ProcessPath(opts)
	if err != nil {
		log.Fatalf("❌ Processing failed: %v", err)
	}

	// Show results
	showSummary(results, opts.Verbose)
}

// showSummary displays processing results - use types.ProcessingResult
func showSummary(results []*types.ProcessingResult, verbose bool) {
	successful := 0
	failed := 0
	skipped := 0
	generated := 0
	transformed := 0
	totalTokens := 0
	var failedFiles []string

	for _, result := range results {
		if result.Skipped {
			skipped++
		} else if result.Success {
			successful++
			totalTokens += result.AITokensUsed
			if result.Mode == types.ModeGenerate {
				generated++
			} else {
				transformed++
			}
		} else {
			failed++
			if result.Error != nil {
				failedFiles = append(failedFiles, fmt.Sprintf("%s: %v", result.InputFile, result.Error))
			}
		}
	}

	fmt.Printf("\n🎯 Summary:\n")
	fmt.Printf("  ✅ Successful: %d", successful)
	if generated > 0 && transformed > 0 {
		fmt.Printf(" (%d generated, %d transformed)", generated, transformed)
	} else if generated > 0 {
		fmt.Printf(" (generated)")
	} else if transformed > 0 {
		fmt.Printf(" (transformed)")
	}
	fmt.Println()

	if skipped > 0 {
		fmt.Printf("  ⏭️  Skipped: %d\n", skipped)
	}
	if failed > 0 {
		fmt.Printf("  ❌ Failed: %d\n", failed)
	}
	fmt.Printf("  🤖 AI Tokens Used: %d\n", totalTokens)

	if failed > 0 && verbose {
		fmt.Printf("\n❌ Failed files:\n")
		for _, failure := range failedFiles {
			fmt.Printf("  - %s\n", failure)
		}
	}

	if successful > 0 {
		fmt.Printf("\n✨ Processing complete!\n")
	}
}

// handleSaveCommand saves current options as a command - use types.Command
func handleSaveCommand(cmdManager *commands.Manager, name string, opts *types.ProcessingOptions) {
	cmd := &types.Command{
		Name:        name,
		Description: fmt.Sprintf("Custom command: %s", name),
		Mode:        opts.Mode,
		Prompt:      opts.AIPrompt,
		PromptFile:  opts.PromptFile,
		Options: types.CommandOptions{
			OutputMode:      string(opts.OutputMode),
			OutputSuffix:    opts.OutputSuffix,
			FilePattern:     opts.FilePattern,
			ExcludePattern:  opts.ExcludePattern,
			ContextPatterns: opts.ContextPatterns,
			ContextFiles:    opts.ContextFiles,
			Recursive:       opts.Recursive,
			RemoveComments:  opts.RemoveComments,
			BackupOriginal:  opts.BackupOriginal,
			Model:           opts.Model,
			Temperature:     opts.Temperature,
			MaxTokens:       opts.MaxTokens,
		},
	}

	if err := cmdManager.SaveCommand(cmd); err != nil {
		log.Fatalf("❌ Failed to save command: %v", err)
	}

	fmt.Printf("✅ Command '%s' saved\n", name)
	fmt.Printf("Usage: presto --cmd %s\n", name)
}

func handleListCommands(cmdManager *commands.Manager) {
	fmt.Println("📋 Available Commands:")
	fmt.Println()

	fmt.Println("Built-in Commands:")
	builtins := cmdManager.GetBuiltinCommands()
	for name, cmd := range builtins {
		fmt.Printf("  %s - %s\n", name, cmd.Description)
	}

	fmt.Println()
	fmt.Println("User Commands:")
	userCmds := cmdManager.GetUserCommands()
	if len(userCmds) == 0 {
		fmt.Println("  (none)")
	} else {
		for name, cmd := range userCmds {
			fmt.Printf("  %s - %s\n", name, cmd.Description)
		}
	}
}

func handleShowCommand(cmdManager *commands.Manager, name string) {
	cmd, err := cmdManager.GetCommand(name)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}

	fmt.Printf("📋 Command: %s\n", cmd.Name)
	fmt.Printf("Description: %s\n", cmd.Description)
	fmt.Printf("Mode: %s\n", cmd.Mode)
	fmt.Printf("Prompt: %s\n", cmd.Prompt)
	if cmd.PromptFile != "" {
		fmt.Printf("Prompt File: %s\n", cmd.PromptFile)
	}

	if len(cmd.Variables) > 0 {
		fmt.Println("Variables:")
		for k, v := range cmd.Variables {
			fmt.Printf("  %s=%s\n", k, v)
		}
	}

	fmt.Printf("Built-in: %t\n", cmdManager.IsBuiltin(name))
}

func handleDeleteCommand(cmdManager *commands.Manager, name string) {
	if err := cmdManager.DeleteCommand(name); err != nil {
		log.Fatalf("❌ Failed to delete command: %v", err)
	}
	fmt.Printf("✅ Command '%s' deleted\n", name)
}

func handleEditCommand(cmdManager *commands.Manager, name string) {
	// Create a template command file for editing
	// templatePath := fmt.Sprintf("%s.yaml", name)

	var cmd *types.Command
	if existingCmd, err := cmdManager.GetCommand(name); err == nil {
		cmd = existingCmd
	} else {
		cmd = cmdManager.GenerateCommandTemplate(name)
	}

	// Write template to file for editing
	// User can then load it with save-command option
	fmt.Printf("Edit template created for command '%s'\n", name)
	fmt.Printf("Modify the template and use --save-command to save changes\n")
	fmt.Printf("Template: %+v\n", cmd)
}

func showHelpText() {
	fmt.Printf(`Presto v%s - AI File Processor

USAGE:
  presto [options]

BASIC OPTIONS:
  --prompt TEXT           AI instruction text
  --cmd NAME             Use predefined command
  --input PATH           File or directory to process
  --recursive            Process directories recursively
  --output MODE          Output mode: inplace|separate|stdout|file
  --dry-run              Preview without making changes

EXAMPLES:
  presto --prompt "Add comments" --input main.go
  presto --cmd add-docs --input . --recursive
  presto --generate --prompt "Create README" --context *.go --output-file README.md

Run 'presto --help-full' for complete options list.
`, version)
}
