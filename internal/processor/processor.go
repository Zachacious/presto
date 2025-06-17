package processor

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Zachacious/presto/internal/ai"
	"github.com/Zachacious/presto/internal/comments"
	"github.com/Zachacious/presto/internal/config"
	"github.com/Zachacious/presto/internal/context"
	"github.com/Zachacious/presto/internal/language"
	"github.com/Zachacious/presto/internal/prompts"
	"github.com/Zachacious/presto/internal/utils"
	"github.com/Zachacious/presto/pkg/types"
)

// Processor handles file processing operations
type Processor struct {
	config         *config.Config
	aiClient       *ai.Client
	commentRemover *comments.Remover
	contextHandler *context.Handler
	promptLoader   *prompts.Loader
}

// New creates a new processor
func New(cfg *config.Config) (*Processor, error) {
	aiClient := ai.New(&cfg.AI)

	if err := aiClient.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("AI configuration invalid: %w", err)
	}

	return &Processor{
		config:         cfg,
		aiClient:       aiClient,
		commentRemover: comments.New(),
		contextHandler: context.New(),
		promptLoader:   prompts.New(),
	}, nil
}

// ProcessPath processes a file or directory based on options
func (p *Processor) ProcessPath(opts *types.ProcessingOptions) ([]*types.ProcessingResult, error) {
	// Load prompt if from file
	if opts.PromptFile != "" {
		if err := p.promptLoader.ValidatePromptFile(opts.PromptFile); err != nil {
			return nil, err
		}
		prompt, err := p.promptLoader.LoadPrompt(opts.PromptFile, nil)
		if err != nil {
			return nil, err
		}
		opts.AIPrompt = prompt
	}

	// Load context files
	var contextFiles []*types.ContextFile
	if len(opts.ContextPatterns) > 0 || len(opts.ContextFiles) > 0 {
		var err error
		contextFiles, err = p.contextHandler.LoadContext(
			opts.ContextPatterns,
			opts.ContextFiles,
			opts.InputPath,
			p.config.Filters.MaxFileSize,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load context: %w", err)
		}

		if opts.Verbose && len(contextFiles) > 0 {
			fmt.Printf("📚 %s\n", p.contextHandler.SummarizeContext(contextFiles))
		}
	}

	// Handle different modes
	switch opts.Mode {
	case types.ModeGenerate:
		return p.processGenerate(opts, contextFiles)
	case types.ModeTransform:
		return p.processTransform(opts, contextFiles)
	default:
		return nil, fmt.Errorf("unknown processing mode: %s", opts.Mode)
	}
}

// processGenerate handles generate mode - creates new files from context
func (p *Processor) processGenerate(opts *types.ProcessingOptions, contextFiles []*types.ContextFile) ([]*types.ProcessingResult, error) {
	if len(contextFiles) == 0 {
		return nil, fmt.Errorf("generate mode requires context files or patterns")
	}

	if opts.OutputPath == "" {
		return nil, fmt.Errorf("generate mode requires output file path (--output-file)")
	}

	if opts.DryRun {
		result := &types.ProcessingResult{
			InputFile:  fmt.Sprintf("context files (%d)", len(contextFiles)),
			OutputFile: opts.OutputPath,
			Success:    true,
			Mode:       types.ModeGenerate,
		}
		return []*types.ProcessingResult{result}, nil
	}

	// Create AI request
	aiReq := types.AIRequest{
		Prompt:      opts.AIPrompt,
		Content:     "", // No target content in generate mode
		Language:    language.DetectLanguage(opts.OutputPath),
		MaxTokens:   p.getMaxTokens(opts),
		Temperature: p.getTemperature(opts),
		Mode:        types.ModeGenerate,
	}

	startTime := time.Now()
	aiResp, err := p.aiClient.ProcessContent(aiReq, contextFiles)
	if err != nil {
		return []*types.ProcessingResult{{
			InputFile:  fmt.Sprintf("context files (%d)", len(contextFiles)),
			OutputFile: opts.OutputPath,
			Success:    false,
			Error:      fmt.Errorf("AI generation failed: %w", err),
			Mode:       types.ModeGenerate,
			Duration:   time.Since(startTime),
		}}, nil
	}

	// Write generated content
	if err := utils.EnsureDir(filepath.Dir(opts.OutputPath)); err != nil {
		return []*types.ProcessingResult{{
			InputFile:  fmt.Sprintf("context files (%d)", len(contextFiles)),
			OutputFile: opts.OutputPath,
			Success:    false,
			Error:      fmt.Errorf("failed to create output directory: %w", err),
			Mode:       types.ModeGenerate,
			Duration:   time.Since(startTime),
		}}, nil
	}

	if err := os.WriteFile(opts.OutputPath, []byte(aiResp.Content), 0644); err != nil {
		return []*types.ProcessingResult{{
			InputFile:  fmt.Sprintf("context files (%d)", len(contextFiles)),
			OutputFile: opts.OutputPath,
			Success:    false,
			Error:      fmt.Errorf("failed to write output file: %w", err),
			Mode:       types.ModeGenerate,
			Duration:   time.Since(startTime),
		}}, nil
	}

	return []*types.ProcessingResult{{
		InputFile:    fmt.Sprintf("context files (%d)", len(contextFiles)),
		OutputFile:   opts.OutputPath,
		Success:      true,
		BytesChanged: len(aiResp.Content),
		AITokensUsed: aiResp.TokensUsed,
		Mode:         types.ModeGenerate,
		Duration:     time.Since(startTime),
	}}, nil
}

// processTransform handles transform mode - modifies existing files
func (p *Processor) processTransform(opts *types.ProcessingOptions, contextFiles []*types.ContextFile) ([]*types.ProcessingResult, error) {
	files, err := p.discoverFiles(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to discover files: %w", err)
	}

	if opts.Verbose {
		fmt.Printf("📁 Found %d files to process\n", len(files))
	}

	if len(files) == 0 {
		return []*types.ProcessingResult{{
			InputFile:  opts.InputPath,
			OutputFile: "",
			Success:    true,
			Skipped:    true,
			SkipReason: "No matching files found",
			Mode:       types.ModeTransform,
		}}, nil
	}

	return p.processFiles(files, opts, contextFiles)
}

// discoverFiles finds all files that should be processed
func (p *Processor) discoverFiles(opts *types.ProcessingOptions) ([]*types.FileInfo, error) {
	var files []*types.FileInfo

	info, err := os.Stat(opts.InputPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		// Single file
		fileInfo, err := p.createFileInfo(opts.InputPath)
		if err != nil {
			return nil, err
		}
		if p.shouldProcessFile(fileInfo, opts) {
			files = append(files, fileInfo)
		}
		return files, nil
	}

	// Directory - walk through files
	walkFunc := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			// Check if directory should be excluded
			if p.shouldExcludeDir(path, opts) {
				return fs.SkipDir
			}
			// Skip if not recursive and not the root directory
			if !opts.Recursive && path != opts.InputPath {
				return fs.SkipDir
			}
			return nil
		}

		fileInfo, err := p.createFileInfo(path)
		if err != nil {
			if opts.Verbose {
				fmt.Printf("⚠️  Warning: %v\n", err)
			}
			return nil // Continue processing other files
		}

		if p.shouldProcessFile(fileInfo, opts) {
			files = append(files, fileInfo)
		}

		return nil
	}

	if err := filepath.WalkDir(opts.InputPath, walkFunc); err != nil {
		return nil, err
	}

	return files, nil
}

// createFileInfo creates a FileInfo from a file path
func (p *Processor) createFileInfo(path string) (*types.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	lang := language.DetectLanguage(path)

	return &types.FileInfo{
		Path:         path,
		OriginalPath: path,
		Language:     lang,
		Size:         info.Size(),
	}, nil
}

// shouldProcessFile determines if a file should be processed
func (p *Processor) shouldProcessFile(file *types.FileInfo, opts *types.ProcessingOptions) bool {
	// Check file size
	if file.Size > p.config.Filters.MaxFileSize {
		return false
	}

	// Check if language is supported (text file)
	if !language.IsTextFile(file.Language) {
		return false
	}

	// Check file pattern
	if opts.FilePattern != "" {
		matched, err := regexp.MatchString(opts.FilePattern, file.Path)
		if err != nil || !matched {
			return false
		}
	}

	// Check exclude pattern
	if opts.ExcludePattern != "" {
		matched, err := regexp.MatchString(opts.ExcludePattern, file.Path)
		if err == nil && matched {
			return false
		}
	}

	// Check include/exclude extensions
	ext := strings.ToLower(filepath.Ext(file.Path))

	if len(p.config.Filters.IncludeExts) > 0 {
		found := false
		for _, allowedExt := range p.config.Filters.IncludeExts {
			if ext == allowedExt {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	for _, excludeExt := range p.config.Filters.ExcludeExts {
		if ext == excludeExt {
			return false
		}
	}

	// Check exclude files
	filename := filepath.Base(file.Path)
	for _, excludeFile := range p.config.Filters.ExcludeFiles {
		if matched, _ := filepath.Match(excludeFile, filename); matched {
			return false
		}
	}

	return true
}

// shouldExcludeDir determines if a directory should be excluded
func (p *Processor) shouldExcludeDir(path string, opts *types.ProcessingOptions) bool {
	dirName := filepath.Base(path)

	for _, excludeDir := range p.config.Filters.ExcludeDirs {
		if dirName == excludeDir {
			return true
		}
	}

	return false
}

// processFiles processes multiple files concurrently
func (p *Processor) processFiles(files []*types.FileInfo, opts *types.ProcessingOptions, contextFiles []*types.ContextFile) ([]*types.ProcessingResult, error) {
	results := make([]*types.ProcessingResult, len(files))

	// Use semaphore to limit concurrent processing
	maxConcurrent := opts.MaxConcurrent
	if maxConcurrent <= 0 {
		maxConcurrent = p.config.Defaults.MaxConcurrent
	}

	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, file := range files {
		wg.Add(1)
		go func(index int, f *types.FileInfo) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			result := p.processFile(f, opts, contextFiles)

			mu.Lock()
			results[index] = result
			mu.Unlock()

			if opts.Verbose {
				if result.Success {
					fmt.Printf("✅ %s\n", result.InputFile)
				} else if result.Skipped {
					fmt.Printf("⏭️  %s: %s\n", result.InputFile, result.SkipReason)
				} else {
					fmt.Printf("❌ %s: %v\n", result.InputFile, result.Error)
				}
			}
		}(i, file)
	}

	wg.Wait()
	return results, nil
}

// processFile processes a single file
func (p *Processor) processFile(file *types.FileInfo, opts *types.ProcessingOptions, contextFiles []*types.ContextFile) *types.ProcessingResult {
	startTime := time.Now()

	result := &types.ProcessingResult{
		InputFile: file.Path,
		Mode:      types.ModeTransform,
		Duration:  time.Since(startTime),
	}

	// Read file content
	content, err := os.ReadFile(file.Path)
	if err != nil {
		result.Error = fmt.Errorf("failed to read file: %w", err)
		return result
	}

	originalContent := string(content)
	processedContent := originalContent

	// Remove comments if requested
	if opts.RemoveComments {
		processedContent = p.commentRemover.RemoveComments(processedContent, file.Language)
	}

	// Handle dry run
	if opts.DryRun {
		result.Success = true
		result.OutputFile = p.generateOutputPath(file.Path, opts)
		result.Duration = time.Since(startTime)
		result.BytesChanged = len(processedContent) - len(originalContent)
		return result
	}

	// Apply AI processing if prompt provided
	if opts.AIPrompt != "" {
		aiReq := types.AIRequest{
			Prompt:      opts.AIPrompt,
			Content:     processedContent,
			Language:    file.Language,
			MaxTokens:   p.getMaxTokens(opts),
			Temperature: p.getTemperature(opts),
			Mode:        types.ModeTransform,
		}

		aiResp, err := p.aiClient.ProcessContent(aiReq, contextFiles)
		if err != nil {
			result.Error = fmt.Errorf("AI processing failed: %w", err)
			return result
		}

		processedContent = aiResp.Content
		result.AITokensUsed = aiResp.TokensUsed
	}

	// Handle output
	outputPath := p.generateOutputPath(file.Path, opts)

	if opts.OutputMode == types.OutputModeStdout {
		fmt.Print(processedContent)
		result.Success = true
		result.OutputFile = "stdout"
	} else {
		// Create backup if needed
		if opts.BackupOriginal && opts.OutputMode == types.OutputModeInPlace {
			backupPath := file.Path + ".backup"
			if err := utils.CopyFile(file.Path, backupPath); err != nil {
				result.Error = fmt.Errorf("failed to create backup: %w", err)
				return result
			}
		}

		// Write processed content
		if err := p.writeFile(outputPath, processedContent); err != nil {
			result.Error = fmt.Errorf("failed to write output: %w", err)
			return result
		}

		result.Success = true
		result.OutputFile = outputPath
	}

	result.Duration = time.Since(startTime)
	result.BytesChanged = len(processedContent) - len(originalContent)

	return result
}

// generateOutputPath generates the output file path based on options
func (p *Processor) generateOutputPath(inputPath string, opts *types.ProcessingOptions) string {
	switch opts.OutputMode {
	case types.OutputModeInPlace:
		return inputPath
	case types.OutputModeFile:
		if opts.OutputPath != "" {
			return opts.OutputPath
		}
		return inputPath + opts.OutputSuffix
	case types.OutputModeSeparate:
		return inputPath + opts.OutputSuffix
	default:
		return inputPath + opts.OutputSuffix
	}
}

// writeFile writes content to a file, creating directories if needed
func (p *Processor) writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := utils.EnsureDir(dir); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// getMaxTokens returns the max tokens to use
func (p *Processor) getMaxTokens(opts *types.ProcessingOptions) int {
	if opts.MaxTokens > 0 {
		return opts.MaxTokens
	}
	return p.config.AI.MaxTokens
}

// getTemperature returns the temperature to use
func (p *Processor) getTemperature(opts *types.ProcessingOptions) float64 {
	if opts.Temperature > 0 {
		return opts.Temperature
	}
	return p.config.AI.Temperature
}
