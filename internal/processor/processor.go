package processor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Zachacious/presto/internal/ai"
	"github.com/Zachacious/presto/internal/comments"
	"github.com/Zachacious/presto/internal/config"
	"github.com/Zachacious/presto/internal/language"
	"github.com/Zachacious/presto/internal/ui"
	"github.com/Zachacious/presto/pkg/types"
)

// Default system prompt focused on complete file output
const DEFAULT_SYSTEM_PROMPT = `
detailed thinking on

CRITICAL INSTRUCTIONS:
- You must return the COMPLETE file content with your changes applied
- Return ONLY the file content - no explanations, no markdown blocks, no commentary
- Do not truncate or summarize - include every line of the original file
- Apply the requested changes but preserve all other content exactly
- Do not add explanatory comments about what you changed
- The response will be written directly to a file, so it must be valid, complete file content

IMPORTANT: Do exactly what is asked and nothing else. Do not add extra features, improvements, or changes beyond the specific request.`

// Processor handles file processing operations
type Processor struct {
	aiClient       *ai.Client
	commentRemover *comments.Remover
	config         *config.Config
	ui             *ui.UI
}

// New creates a new processor
func New(cfg *config.Config) (*Processor, error) {
	// Validate configuration
	if err := config.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create AI client
	aiClient := ai.New(&cfg.AI)
	if err := aiClient.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid AI configuration: %w", err)
	}

	return &Processor{
		aiClient:       aiClient,
		commentRemover: comments.New(),
		config:         cfg,
	}, nil
}

// ProcessPath processes files based on the given options
func (p *Processor) ProcessPath(opts *types.ProcessingOptions) ([]*types.ProcessingResult, error) {
	// Update UI verbose setting
	p.ui = ui.New(opts.Verbose)

	// Load prompt from file if specified
	if opts.PromptFile != "" {
		content, err := os.ReadFile(opts.PromptFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read prompt file: %w", err)
		}
		opts.AIPrompt = string(content)
	}

	// Find files to process
	files, err := p.findFiles(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found to process")
	}

	// Load context files
	contextFiles, err := p.loadContextFiles(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load context files: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("📁 Found %d files to process\n", len(files))
		if len(contextFiles) > 0 {
			fmt.Printf("📋 Loaded %d context files\n", len(contextFiles))
		}
	}

	// Show processing start info
	p.ui.ProcessingStart(len(files), opts.Mode, opts.Model)

	// Process files
	switch opts.Mode {
	case types.ModeGenerate:
		return p.processGenerate(opts, contextFiles)
	case types.ModeTransform:
		return p.processTransform(opts, files, contextFiles)
	default:
		return nil, fmt.Errorf("unknown processing mode: %s", opts.Mode)
	}
}

// processTransform processes files in transform mode
func (p *Processor) processTransform(opts *types.ProcessingOptions, files []*types.FileInfo, contextFiles []*types.ContextFile) ([]*types.ProcessingResult, error) {
	if opts.DryRun {
		return p.simulateTransform(opts, files), nil
	}

	// Create channels for jobs and results
	jobs := make(chan *types.FileInfo, len(files))
	results := make(chan *types.ProcessingResult, len(files))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < opts.MaxConcurrent; i++ {
		wg.Add(1)
		go p.transformWorker(&wg, jobs, results, opts, contextFiles)
	}

	// Send jobs
	for _, file := range files {
		jobs <- file
	}
	close(jobs)

	// Wait for completion and collect results
	go func() {
		wg.Wait()
		close(results)
	}()

	var allResults []*types.ProcessingResult
	completed := 0
	total := len(files)

	for result := range results {
		allResults = append(allResults, result)
		completed++

		// Update progress if verbose
		if opts.Verbose {
			p.ui.Progress(fmt.Sprintf("Progress: %d/%d files completed", completed, total))
		}
	}

	return allResults, nil
}

// transformWorker processes individual files
func (p *Processor) transformWorker(wg *sync.WaitGroup, jobs <-chan *types.FileInfo, results chan<- *types.ProcessingResult, opts *types.ProcessingOptions, contextFiles []*types.ContextFile) {
	defer wg.Done()

	for file := range jobs {
		result := p.processFile(file, opts, contextFiles)
		results <- result
	}
}

func (p *Processor) getSystemPrompt(opts *types.ProcessingOptions) (string, error) {
	// 1. File override takes priority
	if opts.SystemPromptFile != "" {
		content, err := os.ReadFile(opts.SystemPromptFile)
		if err != nil {
			return "", fmt.Errorf("failed to read system prompt file: %w", err)
		}
		return strings.TrimSpace(string(content)), nil
	}

	// 2. Direct override
	if opts.SystemPrompt != "" {
		return opts.SystemPrompt, nil
	}

	// 3. Default
	return DEFAULT_SYSTEM_PROMPT, nil
}

// Update the processFile function to pass the file name
func (p *Processor) processFile(file *types.FileInfo, opts *types.ProcessingOptions, contextFiles []*types.ContextFile) *types.ProcessingResult {
	startTime := time.Now()

	result := &types.ProcessingResult{
		InputFile: file.Path,
		Mode:      opts.Mode,
		Duration:  0,
	}

	// Show processing status
	p.ui.FileProcessing(filepath.Base(file.Path))

	// Read file content
	content, err := os.ReadFile(file.Path)
	if err != nil {
		result.Error = fmt.Errorf("failed to read file: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	contentStr := string(content)

	// Remove comments if requested
	if opts.RemoveComments {
		contentStr = p.commentRemover.RemoveComments(contentStr, file.Language)
	}

	// Get system prompt
	systemPrompt, err := p.getSystemPrompt(opts)
	if err != nil {
		result.Error = fmt.Errorf("failed to get system prompt: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Build final prompt
	var finalPrompt string
	if opts.Mode == types.ModeTransform {
		// For transform: system prompt + user prompt
		finalPrompt = systemPrompt + "\n\n" + opts.AIPrompt
	} else {
		// For generate: just user prompt (no file constraints needed)
		finalPrompt = opts.AIPrompt
	}

	// Process with AI - WITH UI UPDATES
	_, totalTokens, err := p.processWithContinuationAndUI(file, finalPrompt, contentStr, opts, contextFiles)
	if err != nil {
		result.Error = fmt.Errorf("AI processing failed: %w", err)
		result.Duration = time.Since(startTime)
		p.ui.FileError(file.Path, result.Error)
		return result
	}

	result.AITokensUsed = totalTokens

	// Create AI request
	aiReq := types.AIRequest{
		Prompt:      finalPrompt,
		Content:     contentStr,
		FileName:    file.Path,
		Language:    file.Language,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		Mode:        opts.Mode,
	}

	// Process with AI
	aiResp, err := p.aiClient.ProcessContent(aiReq, contextFiles)
	if err != nil {
		result.Error = fmt.Errorf("AI processing failed: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.AITokensUsed = aiResp.TokensUsed

	// Handle output
	outputFile, err := p.handleOutput(file.Path, aiResp.Content, opts)
	if err != nil {
		result.Error = fmt.Errorf("failed to write output: %w", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.OutputFile = outputFile
	result.Success = true
	result.BytesChanged = len(aiResp.Content) - len(contentStr)
	result.Duration = time.Since(startTime)

	// Show success
	p.ui.FileSuccess(file.Path, outputFile, result.Duration, result.AITokensUsed)

	return result
}

// NEW: Process with continuation and UI updates
func (p *Processor) processWithContinuationAndUI(file *types.FileInfo, prompt, originalContent string, opts *types.ProcessingOptions, contextFiles []*types.ContextFile) (string, int, error) {
	const MAX_CONTINUATIONS = 5

	var fullContent strings.Builder
	var totalTokens int
	currentPrompt := prompt

	for attempt := 0; attempt < MAX_CONTINUATIONS; attempt++ {
		// Update UI for continuation attempts
		if attempt > 0 {
			p.ui.FileContinuation(filepath.Base(file.Path), attempt, MAX_CONTINUATIONS)
		}

		// Create AI request
		aiReq := types.AIRequest{
			Prompt:      currentPrompt,
			Content:     originalContent,
			FileName:    file.Path,
			Language:    file.Language,
			MaxTokens:   opts.MaxTokens,
			Temperature: opts.Temperature,
			Mode:        opts.Mode,
		}

		// Process with AI
		aiResp, err := p.aiClient.ProcessContent(aiReq, contextFiles)
		if err != nil {
			return "", totalTokens, err
		}

		totalTokens += aiResp.TokensUsed

		// First response - add everything
		if attempt == 0 {
			fullContent.WriteString(aiResp.Content)
		} else {
			// Continuation - try to merge intelligently
			merged := p.aiClient.MergeContinuation(fullContent.String(), aiResp.Content)
			fullContent.Reset()
			fullContent.WriteString(merged)
		}

		// Check if response is complete
		if aiResp.IsComplete() {
			break
		}

		// For generate mode, don't continue
		if opts.Mode == types.ModeGenerate {
			break
		}

		// Check if we have reasonable completeness
		if attempt > 0 && p.aiClient.LooksReasonablyComplete(fullContent.String(), originalContent, file.Language) {
			break
		}

		// Prepare continuation prompt
		currentPrompt = p.aiClient.BuildContinuationPrompt(fullContent.String(), originalContent, aiReq)

		// If we're on the last attempt, warn the user
		if attempt == MAX_CONTINUATIONS-1 {
			p.ui.FileIncompleteWarning(filepath.Base(file.Path), MAX_CONTINUATIONS)
		}
	}

	return fullContent.String(), totalTokens, nil
}

// processGenerate processes files in generate mode
func (p *Processor) processGenerate(opts *types.ProcessingOptions, contextFiles []*types.ContextFile) ([]*types.ProcessingResult, error) {
	if opts.DryRun {
		result := &types.ProcessingResult{
			InputFile:  "context files",
			OutputFile: opts.OutputPath,
			Success:    true,
			Mode:       types.ModeGenerate,
		}
		return []*types.ProcessingResult{result}, nil
	}

	startTime := time.Now()

	result := &types.ProcessingResult{
		InputFile:  fmt.Sprintf("%d context files", len(contextFiles)),
		OutputFile: opts.OutputPath,
		Mode:       types.ModeGenerate,
	}

	// For generate mode, system prompt is optional
	finalPrompt := opts.AIPrompt
	if opts.SystemPrompt != "" || opts.SystemPromptFile != "" {
		systemPrompt, err := p.getSystemPrompt(opts)
		if err != nil {
			result.Error = fmt.Errorf("failed to get system prompt: %w", err)
			result.Duration = time.Since(startTime)
			return []*types.ProcessingResult{result}, nil
		}
		finalPrompt = systemPrompt + "\n\n" + opts.AIPrompt
	}

	// Create AI request for generation
	aiReq := types.AIRequest{
		Prompt:      finalPrompt,
		Content:     "",
		Language:    types.LangText,
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
		Mode:        opts.Mode,
	}

	// Process with AI
	aiResp, err := p.aiClient.ProcessContent(aiReq, contextFiles)
	if err != nil {
		result.Error = fmt.Errorf("AI processing failed: %w", err)
		result.Duration = time.Since(startTime)
		return []*types.ProcessingResult{result}, nil
	}

	result.AITokensUsed = aiResp.TokensUsed

	// Write output file
	if err := os.WriteFile(opts.OutputPath, []byte(aiResp.Content), 0644); err != nil {
		result.Error = fmt.Errorf("failed to write output file: %w", err)
	} else {
		result.Success = true
		result.BytesChanged = len(aiResp.Content)
	}

	result.Duration = time.Since(startTime)
	return []*types.ProcessingResult{result}, nil
}

// Helper methods (findFiles, loadContextFiles, etc.) remain mostly the same
// but I'll include the key ones:

func (p *Processor) findFiles(opts *types.ProcessingOptions) ([]*types.FileInfo, error) {
	var files []*types.FileInfo

	err := filepath.Walk(opts.InputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories unless we find files in them
		if info.IsDir() {
			// Skip if not recursive and not the root path
			if !opts.Recursive && path != opts.InputPath {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply file filtering
		if p.shouldSkipFile(path, opts) {
			return nil
		}

		// Create file info
		lang := language.DetectLanguage(path)
		fileInfo := &types.FileInfo{
			Path:         path,
			OriginalPath: path,
			Language:     lang,
			Size:         info.Size(),
		}

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

func (p *Processor) shouldSkipFile(path string, opts *types.ProcessingOptions) bool {
	// Check file size limit
	if info, err := os.Stat(path); err == nil {
		if info.Size() > p.config.Filters.MaxFileSize {
			return true
		}
	}

	// Check exclude pattern
	if opts.ExcludePattern != "" {
		if matched, _ := regexp.MatchString(opts.ExcludePattern, path); matched {
			return true
		}
	}

	// Check include pattern
	if opts.FilePattern != "" {
		if matched, _ := regexp.MatchString(opts.FilePattern, path); !matched {
			return true
		}
	}

	// Check extension filters
	ext := strings.ToLower(filepath.Ext(path))
	for _, excludeExt := range p.config.Filters.ExcludeExts {
		if ext == excludeExt {
			return true
		}
	}

	return false
}

func (p *Processor) loadContextFiles(opts *types.ProcessingOptions) ([]*types.ContextFile, error) {
	var contextFiles []*types.ContextFile

	// Load individual context files
	for _, file := range opts.ContextFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		lang := language.DetectLanguage(file)
		contextFile := &types.ContextFile{
			Path:     file,
			Language: lang,
			Content:  string(content),
			Label:    filepath.Base(file),
		}
		contextFiles = append(contextFiles, contextFile)
	}

	// Load files matching context patterns
	for _, pattern := range opts.ContextPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			content, err := os.ReadFile(match)
			if err != nil {
				continue
			}

			lang := language.DetectLanguage(match)
			contextFile := &types.ContextFile{
				Path:     match,
				Language: lang,
				Content:  string(content),
				Label:    filepath.Base(match),
			}
			contextFiles = append(contextFiles, contextFile)
		}
	}

	return contextFiles, nil
}

// simulateTransform simulates transform mode for dry runs
func (p *Processor) simulateTransform(opts *types.ProcessingOptions, files []*types.FileInfo) []*types.ProcessingResult {
	var results []*types.ProcessingResult

	for _, file := range files {
		outputFile := file.Path
		switch opts.OutputMode {
		case types.OutputModeSeparate:
			outputFile = file.Path + opts.OutputSuffix
		case types.OutputModeStdout:
			outputFile = "stdout"
		}

		result := &types.ProcessingResult{
			InputFile:  file.Path,
			OutputFile: outputFile,
			Success:    true,
			Mode:       opts.Mode,
		}
		results = append(results, result)

		if opts.Verbose {
			fmt.Printf("Would process: %s -> %s\n", file.Path, outputFile)
		}
	}

	return results
}

// handleOutput processes the AI response and saves it according to output mode
func (p *Processor) handleOutput(inputFile, content string, opts *types.ProcessingOptions) (string, error) {
	switch opts.OutputMode {
	case types.OutputModeInPlace:
		return p.handleInPlaceOutput(inputFile, content, opts)
	case types.OutputModeDirectory:
		return p.handleDirectoryOutput(inputFile, content, opts)
	case types.OutputModeSeparate:
		return p.handleSeparateOutput(inputFile, content, opts)
	case types.OutputModeFile:
		return p.handleFileOutput(content, opts)
	case types.OutputModeStdout:
		return p.handleStdoutOutput(content)
	case types.OutputModePreview:
		return p.handlePreviewOutput(inputFile, content, opts)
	default:
		return "", fmt.Errorf("unsupported output mode: %s", opts.OutputMode)
	}
}

// handleInPlaceOutput modifies the original file (with backup if requested)
func (p *Processor) handleInPlaceOutput(inputFile, content string, opts *types.ProcessingOptions) (string, error) {
	if opts.DryRun {
		return inputFile + " (dry-run)", nil
	}

	// Create backup if requested
	if opts.BackupOriginal {
		backupFile := inputFile + ".backup"
		if err := p.copyFile(inputFile, backupFile); err != nil {
			return "", fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Write new content to original file
	if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return inputFile, nil
}

// handleDirectoryOutput creates parallel directory structure
func (p *Processor) handleDirectoryOutput(inputFile, content string, opts *types.ProcessingOptions) (string, error) {
	if opts.OutputDir == "" {
		return "", fmt.Errorf("output directory not specified")
	}

	// Get relative path from current working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	relPath, err := filepath.Rel(wd, inputFile)
	if err != nil {
		relPath = filepath.Base(inputFile) // fallback to just filename
	}

	// Create output path in parallel structure
	outputFile := filepath.Join(opts.OutputDir, relPath)

	if opts.DryRun {
		return outputFile + " (dry-run)", nil
	}

	// Create directory structure
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write content
	if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFile, nil
}

// handleSeparateOutput creates new file with smart suffix
func (p *Processor) handleSeparateOutput(inputFile, content string, opts *types.ProcessingOptions) (string, error) {
	var outputFile string

	if opts.SmartSuffix {
		// Insert suffix before extension: main.go → main.presto.go
		outputFile = p.addSmartSuffix(inputFile, opts.OutputSuffix)
	} else {
		// Traditional suffix: main.go → main.go.presto
		outputFile = inputFile + opts.OutputSuffix
	}

	if opts.DryRun {
		return outputFile + " (dry-run)", nil
	}

	// Write content to new file
	if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return outputFile, nil
}

// handleFileOutput writes to a specific output file
func (p *Processor) handleFileOutput(content string, opts *types.ProcessingOptions) (string, error) {
	if opts.OutputPath == "" {
		return "", fmt.Errorf("output file path not specified")
	}

	if opts.DryRun {
		return opts.OutputPath + " (dry-run)", nil
	}

	// Create directory if needed
	outputDir := filepath.Dir(opts.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write content
	if err := os.WriteFile(opts.OutputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return opts.OutputPath, nil
}

// handleStdoutOutput prints content to stdout
func (p *Processor) handleStdoutOutput(content string) (string, error) {
	fmt.Print(content)
	return "(stdout)", nil
}

// handlePreviewOutput shows diff and asks for confirmation
func (p *Processor) handlePreviewOutput(inputFile, newContent string, opts *types.ProcessingOptions) (string, error) {
	// Read original content
	originalContent, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to read original file: %w", err)
	}

	// Show diff (simplified - you could use a proper diff library)
	fmt.Printf("\n=== PREVIEW: %s ===\n", inputFile)
	fmt.Printf("Original length: %d bytes\n", len(originalContent))
	fmt.Printf("New length: %d bytes\n", len(newContent))
	fmt.Printf("\nFirst 500 characters of new content:\n")
	fmt.Printf("---\n")
	if len(newContent) > 500 {
		fmt.Printf("%s...\n", newContent[:500])
	} else {
		fmt.Printf("%s\n", newContent)
	}
	fmt.Printf("---\n")

	// Ask user what to do
	fmt.Printf("\nOptions:\n")
	fmt.Printf("1. Save in-place (replace original)\n")
	fmt.Printf("2. Save with backup (.backup)\n")
	fmt.Printf("3. Save as separate file (.presto)\n")
	fmt.Printf("4. Save to custom file\n")
	fmt.Printf("5. Skip this file\n")
	fmt.Printf("Choice (1-5): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read user input")
	}

	choice := strings.TrimSpace(scanner.Text())
	switch choice {
	case "1":
		return p.handleInPlaceOutput(inputFile, newContent, &types.ProcessingOptions{
			DryRun: opts.DryRun,
		})
	case "2":
		return p.handleInPlaceOutput(inputFile, newContent, &types.ProcessingOptions{
			BackupOriginal: true,
			DryRun:         opts.DryRun,
		})
	case "3":
		return p.handleSeparateOutput(inputFile, newContent, &types.ProcessingOptions{
			OutputSuffix: ".presto",
			SmartSuffix:  true,
			DryRun:       opts.DryRun,
		})
	case "4":
		fmt.Printf("Enter output file path: ")
		if !scanner.Scan() {
			return "", fmt.Errorf("failed to read file path")
		}
		customPath := strings.TrimSpace(scanner.Text())
		return p.handleFileOutput(newContent, &types.ProcessingOptions{
			OutputPath: customPath,
			DryRun:     opts.DryRun,
		})
	case "5":
		return "(skipped)", nil
	default:
		return "", fmt.Errorf("invalid choice: %s", choice)
	}
}

// addSmartSuffix inserts suffix before file extension
func (p *Processor) addSmartSuffix(filename, suffix string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		// No extension, just append suffix
		return filename + suffix
	}

	// Insert suffix before extension
	base := strings.TrimSuffix(filename, ext)
	return base + suffix + ext
}

// copyFile creates a copy of the source file
func (p *Processor) copyFile(src, dst string) error {
	sourceContent, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, sourceContent, 0644)
}
