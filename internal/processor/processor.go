package processor

import (
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
	"github.com/Zachacious/presto/pkg/types"
)

// Processor handles file processing operations
type Processor struct {
	aiClient       *ai.Client
	commentRemover *comments.Remover
	config         *config.Config
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

	// Create worker pool
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

	// Wait for completion
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var allResults []*types.ProcessingResult
	for result := range results {
		allResults = append(allResults, result)
		if opts.Verbose {
			if result.Success {
				fmt.Printf("✅ %s -> %s\n", result.InputFile, result.OutputFile)
			} else {
				fmt.Printf("❌ %s: %v\n", result.InputFile, result.Error)
			}
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

// processFile processes a single file
func (p *Processor) processFile(file *types.FileInfo, opts *types.ProcessingOptions, contextFiles []*types.ContextFile) *types.ProcessingResult {
	startTime := time.Now()

	result := &types.ProcessingResult{
		InputFile: file.Path,
		Mode:      opts.Mode,
		Duration:  0,
	}

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

	// Create AI request
	aiReq := types.AIRequest{
		Prompt:      opts.AIPrompt,
		Content:     contentStr,
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

	return result
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

	// Create AI request for generation
	aiReq := types.AIRequest{
		Prompt:      opts.AIPrompt,
		Content:     "", // No specific content for generation
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

// handleOutput writes processed content to the appropriate destination
func (p *Processor) handleOutput(inputFile, content string, opts *types.ProcessingOptions) (string, error) {
	switch opts.OutputMode {
	case types.OutputModeStdout:
		fmt.Print(content)
		return "stdout", nil

	case types.OutputModeInPlace:
		// Create backup if requested
		if opts.BackupOriginal {
			backupFile := inputFile + ".backup"
			if err := p.copyFile(inputFile, backupFile); err != nil {
				return "", fmt.Errorf("failed to create backup: %w", err)
			}
		}

		if err := os.WriteFile(inputFile, []byte(content), 0644); err != nil {
			return "", err
		}
		return inputFile, nil

	case types.OutputModeSeparate:
		outputFile := inputFile + opts.OutputSuffix
		if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
			return "", err
		}
		return outputFile, nil

	default:
		return "", fmt.Errorf("unsupported output mode: %s", opts.OutputMode)
	}
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

func (p *Processor) copyFile(src, dst string) error {
	content, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, content, 0644)
}
