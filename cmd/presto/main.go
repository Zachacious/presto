package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	types "github.com/Zachacious/presto/commands"
	"github.com/Zachacious/presto/internal/commands"
	"github.com/Zachacious/presto/internal/config"
	"github.com/Zachacious/presto/internal/processor"
)

const version = "1.0.0"
const banner = `🎩 presto v%s - Transform your files like magic!`

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ", ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type varFlags map[string]string

func (v varFlags) String() string {
	var parts []string
	for k, val := range v {
		parts = append(parts, fmt.Sprintf("%s=%s", k, val))
	}
	return strings.Join(parts, ", ")
}

func (v varFlags) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("variables must be in format KEY=VALUE")
	}
	v[parts[0]] = parts[1]
	return nil
}

func main() {
	// Command management flags
	var (
		useCommand    = flag.String("cmd", "", "Use a prefab command")
		listCommands  = flag.Bool("list-commands", false, "List all available commands")
		saveCommand   = flag.String("save-command", "", "Save current options as a command")
		deleteCommand = flag.String("delete-command", "", "Delete a saved command")
		showCommand   = flag.String("show-command", "", "Show details of a command")
		editCommand   = flag.String("edit-command", "", "Create/edit a command interactively")
		variables     = make(varFlags)
	)

	// Processing flags
	var (
		promptText      = flag.String("prompt", "", "AI prompt to apply")
		promptFile      = flag.String("prompt-file", "", "Load AI prompt from file")
		inputPath       = flag.String("input", ".", "Input file or directory")
		outputPath      = flag.String("output-file", "", "Output file (for generate mode)")
		outputMode      = flag.String("output", "separate", "Output mode: inplace, separate, stdout, file")
		outputSuffix    = flag.String("suffix", ".presto", "Suffix for separate output files")
		recursive       = flag.Bool("recursive", false, "Process directories recursively")
		filePattern     = flag.String("pattern", "", "Pattern for files to include")
		excludePattern  = flag.String("exclude", "", "Pattern for files to exclude")
		contextFiles    arrayFlags
		contextPatterns arrayFlags
		removeComments  = flag.Bool("remove-comments", false, "Remove comments before processing")
		dryRun          = flag.Bool("dry-run", false, "Show what would be processed")
		verbose         = flag.Bool("verbose", false, "Verbose output")
		maxConcurrent   = flag.Int("concurrent", 3, "Maximum concurrent processing")
		backupOriginal  = flag.Bool("backup", false, "Create backup when using inplace mode")
		model           = flag.String("model", "", "AI model to use")
		temperature     = flag.Float64("temperature", 0, "AI temperature (0.0-2.0)")
		maxTokens       = flag.Int("max-tokens", 0, "Maximum AI tokens")
	)

	// Utility flags
	var (
		showVersion  = flag.Bool("version", false, "Show version")
		showHelp     = flag.Bool("help", false, "Show help")
		listModels   = flag.Bool("list-models", false, "List available AI models")
		generateMode = flag.Bool("generate", false, "Generate new file from context (instead of transform)")
		listPrompts  = flag.Bool("list-prompts", false, "List built-in prompt templates")
		savePrompt   = flag.String("save-prompt", "", "Save built-in prompt to file: name:path")
		configPath   = flag.String("config", "", "Configuration file path")
	)

	flag.Var(&contextFiles, "context", "Context files (can be repeated)")
	flag.Var(&contextPatterns, "context-pattern", "Context file patterns (can be repeated)")
	flag.Var(&variables, "var", "Variables for command substitution KEY=VALUE (can be repeated)")

	flag.Parse()

	// Handle utility commands first
	if *showVersion {
		fmt.Printf(banner+"\n", version)
		return
	}

	if *showHelp {
		showUsage()
		return
	}

	if *listModels {
		showModels()
		return
	}

	if *listPrompts {
		showPrompts()
		return
	}

	if *savePrompt != "" {
		handleSavePrompt(*savePrompt)
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Initialize command manager
	cmdManager, err := commands.New()
	if err != nil {
		log.Fatalf("❌ Failed to initialize commands: %v", err)
	}

	// Handle command management
	if *listCommands {
		showCommands(cmdManager)
		return
	}

	if *showCommand != "" {
		showCommandDetails(cmdManager, *showCommand)
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

	// Build processing options
	opts := &types.ProcessingOptions{
		Mode:            types.ModeTransform,
		AIPrompt:        *promptText,
		PromptFile:      *promptFile,
		InputPath:       *inputPath,
		OutputPath:      *outputPath,
		OutputMode:      types.OutputMode(*outputMode),
		ContextFiles:    contextFiles,
		ContextPatterns: contextPatterns,
		Recursive:       *recursive,
		FilePattern:     *filePattern,
		ExcludePattern:  *excludePattern,
		RemoveComments:  *removeComments,
		DryRun:          *dryRun,
		Verbose:         *verbose,
		MaxConcurrent:   *maxConcurrent,
		BackupOriginal:  *backupOriginal,
		OutputSuffix:    *outputSuffix,
		Model:           *model,
		Temperature:     *temperature,
		MaxTokens:       *maxTokens,
	}

	// Set mode
	if *generateMode {
		opts.Mode = types.ModeGenerate
	}

	// Apply prefab command if specified
	if *useCommand != "" {
		cmd, err := cmdManager.GetCommand(*useCommand)
		if err != nil {
			log.Fatalf("❌ %v", err)
		}

		// Substitute variables
		cmdManager.SubstituteVariables(cmd, variables)

		// Apply command to options
		if err := cmdManager.ApplyCommand(*useCommand, opts); err != nil {
			log.Fatalf("❌ Failed to apply command: %v", err)
		}

		if *verbose {
			fmt.Printf("📋 Using command: %s - %s\n", cmd.Name, cmd.Description)
		}
	}

	// Save command if requested
	if *saveCommand != "" {
		handleSaveCommand(cmdManager, *saveCommand, opts)
		return
	}

	// Validate required parameters
	if opts.AIPrompt == "" && opts.PromptFile == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: Either --prompt, --prompt-file, or --cmd is required\n")
		showUsage()
		os.Exit(1)
	}

	// Validate output mode
	if !isValidOutputMode(string(opts.OutputMode)) {
		log.Fatalf("❌ Invalid output mode: %s", opts.OutputMode)
	}

	// Override model if specified
	if opts.Model != "" {
		cfg.AI.Model = opts.Model
	}
	if opts.Temperature != 0 {
		cfg.AI.Temperature = opts.Temperature
	}
	if opts.MaxTokens != 0 {
		cfg.AI.MaxTokens = opts.MaxTokens
	}

	// Create processor
	proc, err := processor.New(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create processor: %v", err)
	}

	// Show processing info
	if *verbose {
		showProcessingInfo(opts, cfg)
	}

	// Process files
	results, err := proc.ProcessPath(opts)
	if err != nil {
		log.Fatalf("❌ Processing failed: %v", err)
	}

	// Show summary
	showSummary(results, *verbose)
}

func isValidOutputMode(mode string) bool {
	validModes := []string{"inplace", "separate", "stdout", "file"}
	for _, valid := range validModes {
		if mode == valid {
			return true
		}
	}
	return false
}

func showCommands(cmdManager *commands.Manager) {
	fmt.Println("📋 Available Commands:")
	fmt.Println()

	builtins := cmdManager.GetBuiltinCommands()
	user := cmdManager.GetUserCommands()

	if len(builtins) > 0 {
		fmt.Println("🔧 Built-in Commands:")
		for name, cmd := range builtins {
			fmt.Printf("  %-15s %s\n", name, cmd.Description)
		}
		fmt.Println()
	}

	if len(user) > 0 {
		fmt.Println("👤 User Commands:")
		for name, cmd := range user {
			fmt.Printf("  %-15s %s\n", name, cmd.Description)
		}
		fmt.Println()
	}

	fmt.Println("Usage:")
	fmt.Println("  presto --cmd COMMAND_NAME [options]")
	fmt.Println("  presto --show-command COMMAND_NAME")
}

func showCommandDetails(cmdManager *commands.Manager, name string) {
	cmd, err := cmdManager.GetCommand(name)
	if err != nil {
		log.Fatalf("❌ %v", err)
	}

	fmt.Printf("📋 Command: %s\n", cmd.Name)
	fmt.Printf("Description: %s\n", cmd.Description)
	fmt.Printf("Mode: %s\n", cmd.Mode)

	isBuiltin := cmdManager.IsBuiltin(name)
	if isBuiltin {
		fmt.Printf("Type: Built-in\n")
	} else {
		fmt.Printf("Type: User-defined\n")
	}
	fmt.Println()

	if cmd.Prompt != "" {
		fmt.Printf("Prompt:\n%s\n\n", cmd.Prompt)
	}
	if cmd.PromptFile != "" {
		fmt.Printf("Prompt File: %s\n\n", cmd.PromptFile)
	}

	fmt.Println("Options:")
	if cmd.Options.OutputMode != "" {
		fmt.Printf("  Output Mode: %s\n", cmd.Options.OutputMode)
	}
	if cmd.Options.OutputSuffix != "" {
		fmt.Printf("  Output Suffix: %s\n", cmd.Options.OutputSuffix)
	}
	if cmd.Options.FilePattern != "" {
		fmt.Printf("  File Pattern: %s\n", cmd.Options.FilePattern)
	}
	if cmd.Options.ExcludePattern != "" {
		fmt.Printf("  Exclude Pattern: %s\n", cmd.Options.ExcludePattern)
	}
	if len(cmd.Options.ContextPatterns) > 0 {
		fmt.Printf("  Context Patterns: %v\n", cmd.Options.ContextPatterns)
	}
	if len(cmd.Options.ContextFiles) > 0 {
		fmt.Printf("  Context Files: %v\n", cmd.Options.ContextFiles)
	}
	if cmd.Options.Recursive {
		fmt.Printf("  Recursive: true\n")
	}
	if cmd.Options.RemoveComments {
		fmt.Printf("  Remove Comments: true\n")
	}
	if cmd.Options.BackupOriginal {
		fmt.Printf("  Backup Original: true\n")
	}

	if len(cmd.Variables) > 0 {
		fmt.Println("\nVariables:")
		for key, value := range cmd.Variables {
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	fmt.Println("\nUsage:")
	fmt.Printf("  presto --cmd %s [options]\n", cmd.Name)
	if len(cmd.Variables) > 0 {
		fmt.Printf("  presto --cmd %s", cmd.Name)
		for key := range cmd.Variables {
			fmt.Printf(" --var %s=value", key)
		}
		fmt.Println()
	}
}

func handleDeleteCommand(cmdManager *commands.Manager, name string) {
	if cmdManager.IsBuiltin(name) {
		log.Fatalf("❌ Cannot delete built-in command: %s", name)
	}

	if err := cmdManager.DeleteCommand(name); err != nil {
		log.Fatalf("❌ Failed to delete command: %v", err)
	}
	fmt.Printf("✅ Command '%s' deleted\n", name)
}

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

func handleEditCommand(cmdManager *commands.Manager, name string) {
	cmd := cmdManager.GenerateCommandTemplate(name)
	if err := cmdManager.SaveCommand(cmd); err != nil {
		log.Fatalf("❌ Failed to create command template: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	fmt.Printf("✅ Command template '%s' created\n", name)
	fmt.Printf("Edit: %s/.presto/commands/%s.yaml\n", homeDir, name)
}

func handleSavePrompt(savePrompt string) {
	// This would use the prompts loader, but keeping simple for now
	fmt.Printf("❌ Save prompt feature not implemented in this version\n")
	fmt.Printf("Use --list-prompts to see available templates\n")
}

func showPrompts() {
	fmt.Println("🎭 Built-in Prompt Templates:")
	fmt.Println()
	fmt.Println("  add-docs           Add comprehensive documentation")
	fmt.Println("  add-error-handling Add robust error handling")
	fmt.Println("  optimize           Optimize for performance and readability")
	fmt.Println("  modernize          Update to current best practices")
	fmt.Println("  refactor           Improve maintainability")
	fmt.Println("  add-tests          Generate comprehensive tests")
	fmt.Println("  convert-language   Convert to different language")
	fmt.Println()
	fmt.Println("Use these with built-in commands:")
	fmt.Println("  presto --cmd add-docs --recursive")
	fmt.Println("  presto --cmd js2ts --input src/")
}

func showProcessingInfo(opts *types.ProcessingOptions, cfg *config.Config) {
	fmt.Printf("🎩 Processing with presto:\n")
	fmt.Printf("  Mode: %s\n", opts.Mode)
	fmt.Printf("  Input: %s\n", opts.InputPath)
	if opts.OutputPath != "" {
		fmt.Printf("  Output File: %s\n", opts.OutputPath)
	} else {
		fmt.Printf("  Output Mode: %s\n", opts.OutputMode)
	}
	fmt.Printf("  AI Model: %s\n", cfg.AI.Model)
	if len(opts.ContextFiles) > 0 || len(opts.ContextPatterns) > 0 {
		fmt.Printf("  Context: %d files, %d patterns\n", len(opts.ContextFiles), len(opts.ContextPatterns))
	}
	if opts.FilePattern != "" {
		fmt.Printf("  File Pattern: %s\n", opts.FilePattern)
	}
	if opts.ExcludePattern != "" {
		fmt.Printf("  Exclude Pattern: %s\n", opts.ExcludePattern)
	}
	fmt.Printf("  Recursive: %v\n", opts.Recursive)
	fmt.Printf("  Dry Run: %v\n", opts.DryRun)
	fmt.Println()
}

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
		fmt.Printf("\n🎉 Presto! Your files have been magically transformed!\n")
	}
}

func showUsage() {
	fmt.Printf(banner+"\n\n", version)
	fmt.Printf(`Usage:
  presto [options]
  presto --cmd COMMAND_NAME [options]

🎭 Command Management:
  --cmd NAME               Use a prefab command
  --list-commands         List all available commands  
  --show-command NAME     Show command details
  --save-command NAME     Save current options as a command
  --delete-command NAME   Delete a saved command
  --edit-command NAME     Create command template

✨ Processing Options:
  --prompt TEXT           AI prompt to apply
  --prompt-file FILE      Load AI prompt from file
  --input PATH            Input file or directory (default ".")
  --output-file PATH      Output file for generate mode
  --output MODE           Output mode: inplace, separate, stdout, file
  --suffix TEXT           Suffix for separate files (default ".presto")

🎯 Targeting:
  --pattern REGEX         Include files matching pattern
  --exclude REGEX         Exclude files matching pattern
  --recursive            Process directories recursively
  --generate             Generate new file (instead of transform)

🔍 Context:
  --context FILE          Context files (repeatable)
  --context-pattern REGEX Context file patterns (repeatable)

🛠️  Processing:
  --remove-comments       Remove comments before processing
  --concurrent N          Max concurrent processing (default 3)
  --backup               Create backup for inplace mode

🤖 AI Options:
  --model NAME           AI model to use
  --temperature N        AI temperature 0.0-2.0
  --max-tokens N         Maximum AI tokens

🔧 Variables & Config:
  --var KEY=VALUE        Variables for command substitution
  --config FILE          Configuration file path

ℹ️  Utility:
  --dry-run              Show what would be processed
  --verbose              Verbose output
  --list-models          List available AI models
  --list-prompts         List built-in prompt templates
  --version              Show version
  --help                 Show this help

🌟 Examples:

  # Use built-in commands
  presto --cmd add-docs --recursive
  presto --cmd js2ts --input src/

  # Generate new service from patterns
  presto --generate --prompt "Create user service" \
         --context-pattern "*service*.go" \
         --output-file user-service.go

  # Transform with context
  presto --prompt "Refactor to match style" \
         --context style-guide.md \
         --recursive --pattern "\.go$"

  # Create and use custom commands  
  presto --prompt "Add logging" --save-command add-logs
  presto --cmd add-logs

Environment:
  OPENROUTER_API_KEY     Your OpenRouter API key (required)
`)
}

func showModels() {
	fmt.Println("🤖 Popular AI Models (via OpenRouter):")
	fmt.Println()

	models := []struct {
		name        string
		description string
	}{
		{"anthropic/claude-3.5-sonnet", "Best for complex code analysis and generation"},
		{"anthropic/claude-3-haiku", "Fast and efficient for simple tasks"},
		{"openai/gpt-4", "Excellent overall performance"},
		{"openai/gpt-4-turbo", "Latest GPT-4 with extended context"},
		{"openai/gpt-3.5-turbo", "Good balance of speed and capability"},
		{"google/gemini-pro", "Google's flagship model"},
		{"mistralai/mixtral-8x7b-instruct", "Open source, great performance"},
		{"meta-llama/llama-3-70b-instruct", "Meta's latest large model"},
		{"cohere/command-r-plus", "Optimized for reasoning tasks"},
	}

	for _, model := range models {
		fmt.Printf("  %-35s %s\n", model.name, model.description)
	}

	fmt.Println()
	fmt.Println("Visit https://openrouter.ai/docs for the complete list")
	fmt.Println("Set with: --model MODEL_NAME")
}
