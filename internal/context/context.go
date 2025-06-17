package context

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Zachacious/presto/internal/language"
	"github.com/Zachacious/presto/pkg/types"
)

// Handler manages context files and patterns
type Handler struct{}

// New creates a new context handler
func New() *Handler {
	return &Handler{}
}

// LoadContext loads context files from patterns and specific files
func (h *Handler) LoadContext(patterns []string, files []string, basePath string, maxFileSize int64) ([]*types.ContextFile, error) {
	var contextFiles []*types.ContextFile
	seenFiles := make(map[string]bool) // Avoid duplicates

	// Load from patterns
	for _, pattern := range patterns {
		matched, err := h.findFilesByPattern(pattern, basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to match pattern %s: %w", pattern, err)
		}

		for _, file := range matched {
			if seenFiles[file] {
				continue
			}
			seenFiles[file] = true

			contextFile, err := h.loadContextFile(file, maxFileSize)
			if err != nil {
				continue // Skip problematic files
			}
			contextFiles = append(contextFiles, contextFile)
		}
	}

	// Load specific files
	for _, file := range files {
		resolvedPath := h.resolvePath(file, basePath)

		if seenFiles[resolvedPath] {
			continue
		}
		seenFiles[resolvedPath] = true

		contextFile, err := h.loadContextFile(resolvedPath, maxFileSize)
		if err != nil {
			return nil, fmt.Errorf("failed to load context file %s: %w", file, err)
		}
		contextFiles = append(contextFiles, contextFile)
	}

	return contextFiles, nil
}

// findFilesByPattern finds files matching a pattern
func (h *Handler) findFilesByPattern(pattern, basePath string) ([]string, error) {
	var matches []string

	// Handle different pattern types
	if strings.Contains(pattern, "*") || strings.Contains(pattern, "?") {
		// Glob pattern
		globPattern := filepath.Join(basePath, pattern)
		if !filepath.IsAbs(globPattern) && basePath != "." {
			globPattern = filepath.Join(".", pattern)
		}

		globMatches, err := filepath.Glob(globPattern)
		if err != nil {
			return nil, err
		}
		matches = append(matches, globMatches...)

		// Also try recursive glob if pattern doesn't start with */
		if !strings.HasPrefix(pattern, "*/") && !strings.HasPrefix(pattern, "**/") {
			recursivePattern := filepath.Join(basePath, "**", pattern)
			if !filepath.IsAbs(recursivePattern) {
				recursivePattern = filepath.Join(".", "**", pattern)
			}

			// Walk directories for recursive matching
			err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // Continue on errors
				}

				if info.IsDir() {
					if h.shouldSkipDir(info.Name()) {
						return filepath.SkipDir
					}
					return nil
				}

				// Check if file matches pattern
				if matched, _ := filepath.Match(pattern, info.Name()); matched {
					if h.isTextFile(path) {
						matches = append(matches, path)
					}
				}

				return nil
			})
		}
	} else {
		// Regex pattern - search recursively
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", err)
		}

		err = filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Continue on errors
			}

			if info.IsDir() {
				// Skip common ignore directories
				if h.shouldSkipDir(info.Name()) {
					return filepath.SkipDir
				}
				return nil
			}

			if regex.MatchString(path) || regex.MatchString(info.Name()) {
				if h.isTextFile(path) {
					matches = append(matches, path)
				}
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	// Remove duplicates and filter for text files only
	var textFiles []string
	seen := make(map[string]bool)

	for _, match := range matches {
		if seen[match] {
			continue
		}
		seen[match] = true

		if h.isTextFile(match) {
			textFiles = append(textFiles, match)
		}
	}

	return textFiles, nil
}

// loadContextFile loads a single context file
func (h *Handler) loadContextFile(path string, maxFileSize int64) (*types.ContextFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if info.Size() > maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", info.Size(), maxFileSize)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lang := language.DetectLanguage(path)

	return &types.ContextFile{
		Path:     path,
		Language: lang,
		Content:  string(content),
		Label:    h.generateLabel(path),
	}, nil
}

// generateLabel creates a readable label for the context file
func (h *Handler) generateLabel(path string) string {
	// Get relative path from current directory
	if rel, err := filepath.Rel(".", path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return filepath.Base(path)
}

// resolvePath resolves a file path relative to base path
func (h *Handler) resolvePath(file, basePath string) string {
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(basePath, file)
}

// isTextFile determines if a file should be processed as text
func (h *Handler) isTextFile(path string) bool {
	lang := language.DetectLanguage(path)
	return language.IsTextFile(lang)
}

// shouldSkipDir determines if a directory should be skipped during walking
func (h *Handler) shouldSkipDir(name string) bool {
	skipDirs := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor", "venv", ".venv",
		".idea", ".vscode", ".vs",
		"dist", "build", "target", "bin", "obj",
		".next", ".nuxt", "__pycache__",
		"coverage", ".coverage", ".nyc_output",
	}

	for _, skip := range skipDirs {
		if name == skip {
			return true
		}
	}

	return false
}

// ParseContextArguments parses context arguments that might include labels
// Format: "path/to/file.go" or "label:path/to/file.go"
func (h *Handler) ParseContextArguments(args []string) ([]string, map[string]string) {
	var paths []string
	labels := make(map[string]string)

	for _, arg := range args {
		if idx := strings.Index(arg, ":"); idx > 0 && !strings.Contains(arg[:idx], "/") && !strings.Contains(arg[:idx], "\\") {
			// Only treat as label:path if the part before : doesn't contain path separators
			label := arg[:idx]
			path := arg[idx+1:]
			paths = append(paths, path)
			labels[path] = label
		} else {
			paths = append(paths, arg)
		}
	}

	return paths, labels
}

// ApplyLabels applies custom labels to context files
func (h *Handler) ApplyLabels(contextFiles []*types.ContextFile, labels map[string]string) {
	for _, file := range contextFiles {
		if label, exists := labels[file.Path]; exists {
			file.Label = label
		}
	}
}

// SummarizeContext returns a summary of loaded context files
func (h *Handler) SummarizeContext(contextFiles []*types.ContextFile) string {
	if len(contextFiles) == 0 {
		return "No context files loaded"
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Loaded %d context files:\n", len(contextFiles)))

	langCounts := make(map[types.Language]int)
	totalSize := int64(0)

	for _, file := range contextFiles {
		langCounts[file.Language]++
		totalSize += int64(len(file.Content))
	}

	for lang, count := range langCounts {
		summary.WriteString(fmt.Sprintf("  - %s: %d files\n", lang, count))
	}

	summary.WriteString(fmt.Sprintf("  - Total size: %d bytes", totalSize))

	return summary.String()
}
