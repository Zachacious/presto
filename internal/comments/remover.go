package comments

import (
	"regexp"
	"strings"

	"github.com/Zachacious/presto/internal/language"
	"github.com/Zachacious/presto/pkg/types"
)

// Remover handles comment removal from source code
type Remover struct{}

// New creates a new comment remover
func New() *Remover {
	return &Remover{}
}

// RemoveComments removes comments from source code based on language
func (r *Remover) RemoveComments(content string, lang types.Language) string {
	if !language.IsTextFile(lang) {
		return content
	}

	style := language.GetCommentStyle(lang)
	return r.removeCommentsByStyle(content, style)
}

// GetCommentPatterns returns regex patterns for different comment styles
func (r *Remover) GetCommentPatterns(style types.CommentStyle) []string {
	var patterns []string

	// Line comments
	if style.LineComment != "" {
		// Escape special regex characters
		escaped := regexp.QuoteMeta(style.LineComment)
		patterns = append(patterns, escaped+`.*$`)
	}

	// Block comments
	if style.BlockStart != "" && style.BlockEnd != "" {
		start := regexp.QuoteMeta(style.BlockStart)
		end := regexp.QuoteMeta(style.BlockEnd)
		patterns = append(patterns, start+`[\s\S]*?`+end)
	}

	return patterns
}

// removeCommentsByStyle removes comments using the specified style
func (r *Remover) removeCommentsByStyle(content string, style types.CommentStyle) string {
	patterns := r.GetCommentPatterns(style)
	result := content

	for _, pattern := range patterns {
		// Compile with multiline flag for line comments
		re := regexp.MustCompile(`(?m)` + pattern)
		result = re.ReplaceAllString(result, "")
	}

	// Clean up extra whitespace
	result = r.cleanupWhitespace(result)

	return result
}

// cleanupWhitespace removes excessive blank lines
func (r *Remover) cleanupWhitespace(content string) string {
	// Remove multiple consecutive empty lines
	re := regexp.MustCompile(`\n\s*\n\s*\n`)
	content = re.ReplaceAllString(content, "\n\n")

	// Remove trailing whitespace from lines
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	return strings.Join(lines, "\n")
}

// PreserveComments returns content with comments preserved
func (r *Remover) PreserveComments(content string, lang types.Language) string {
	// For now, just return original content
	// This could be extended to extract and format comments differently
	return content
}
