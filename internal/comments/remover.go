package comments

import (
	"regexp"
	"strings"

	"github.com/Zachacious/presto/internal/language"
	"github.com/Zachacious/presto/pkg/types"
)

// Remover handles intelligent comment removal
type Remover struct{}

// New creates a new comment remover
func New() *Remover {
	return &Remover{}
}

// RemoveComments removes comments from content based on language
func (r *Remover) RemoveComments(content string, lang types.Language) string {
	style := language.GetCommentStyle(lang)

	if style.SingleLine == "" && style.MultiStart == "" {
		return content // No comment style defined
	}

	switch lang {
	case types.LangGo, types.LangJavaScript, types.LangTypeScript, types.LangJava, types.LangC, types.LangCPP, types.LangRust:
		return r.removeCStyleComments(content, style)
	case types.LangPython:
		return r.removePythonComments(content, style)
	case types.LangShell, types.LangYAML:
		return r.removeHashComments(content)
	case types.LangHTML:
		return r.removeHTMLComments(content)
	case types.LangCSS:
		return r.removeCSSComments(content)
	case types.LangSQL:
		return r.removeSQLComments(content)
	default:
		return content
	}
}

// removeCStyleComments removes C-style comments (// and /* */)
func (r *Remover) removeCStyleComments(content string, style types.CommentStyle) string {
	lines := strings.Split(content, "\n")
	var result []string
	inMultiComment := false

	for _, line := range lines {
		cleaned := r.processCStyleLine(line, style, &inMultiComment)
		if strings.TrimSpace(cleaned) != "" || !inMultiComment {
			result = append(result, cleaned)
		}
	}

	return strings.Join(result, "\n")
}

func (r *Remover) processCStyleLine(line string, style types.CommentStyle, inMultiComment *bool) string {
	if *inMultiComment {
		if idx := strings.Index(line, style.MultiEnd); idx != -1 {
			*inMultiComment = false
			return r.processCStyleLine(line[idx+len(style.MultiEnd):], style, inMultiComment)
		}
		return "" // Entire line is in multi-line comment
	}

	// Check for single line comments
	if style.SingleLine != "" {
		if idx := strings.Index(line, style.SingleLine); idx != -1 {
			// Make sure it's not inside a string literal
			if !r.isInString(line, idx) {
				line = line[:idx]
			}
		}
	}

	// Check for multi-line comment start
	if style.MultiStart != "" {
		if idx := strings.Index(line, style.MultiStart); idx != -1 {
			if !r.isInString(line, idx) {
				before := line[:idx]
				after := line[idx+len(style.MultiStart):]

				// Check if comment ends on same line
				if endIdx := strings.Index(after, style.MultiEnd); endIdx != -1 {
					return before + r.processCStyleLine(after[endIdx+len(style.MultiEnd):], style, inMultiComment)
				} else {
					*inMultiComment = true
					return strings.TrimRight(before, " \t")
				}
			}
		}
	}

	return line
}

// isInString checks if position is inside a string literal (basic check)
func (r *Remover) isInString(line string, pos int) bool {
	quotes := 0
	doubleQuotes := 0

	for i := 0; i < pos; i++ {
		switch line[i] {
		case '\'':
			if i == 0 || line[i-1] != '\\' {
				quotes++
			}
		case '"':
			if i == 0 || line[i-1] != '\\' {
				doubleQuotes++
			}
		}
	}

	return quotes%2 == 1 || doubleQuotes%2 == 1
}

// removePythonComments removes Python-style comments
func (r *Remover) removePythonComments(content string, style types.CommentStyle) string {
	lines := strings.Split(content, "\n")
	var result []string
	inDocstring := false
	docstringType := ""

	for _, line := range lines {
		cleaned, newInDocstring, newDocstringType := r.processPythonLine(line, inDocstring, docstringType)
		inDocstring = newInDocstring
		docstringType = newDocstringType

		if strings.TrimSpace(cleaned) != "" || !inDocstring {
			result = append(result, cleaned)
		}
	}

	return strings.Join(result, "\n")
}

func (r *Remover) processPythonLine(line string, inDocstring bool, docstringType string) (string, bool, string) {
	if inDocstring {
		if idx := strings.Index(line, docstringType); idx != -1 {
			return line[idx+len(docstringType):], false, ""
		}
		return "", true, docstringType // Still in docstring
	}

	// Check for docstring start
	if idx := strings.Index(line, `"""`); idx != -1 {
		before := line[:idx]
		after := line[idx+3:]
		if endIdx := strings.Index(after, `"""`); endIdx != -1 {
			// Single line docstring
			return before + after[endIdx+3:], false, ""
		} else {
			return strings.TrimRight(before, " \t"), true, `"""`
		}
	}

	if idx := strings.Index(line, `'''`); idx != -1 {
		before := line[:idx]
		after := line[idx+3:]
		if endIdx := strings.Index(after, `'''`); endIdx != -1 {
			// Single line docstring
			return before + after[endIdx+3:], false, ""
		} else {
			return strings.TrimRight(before, " \t"), true, `'''`
		}
	}

	// Check for regular comments
	if idx := strings.Index(line, "#"); idx != -1 {
		if !r.isInString(line, idx) {
			line = line[:idx]
		}
	}

	return line, false, ""
}

// removeHashComments removes hash-style comments (#)
func (r *Remover) removeHashComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}
		result = append(result, strings.TrimRight(line, " \t"))
	}

	return strings.Join(result, "\n")
}

// removeHTMLComments removes HTML comments
func (r *Remover) removeHTMLComments(content string) string {
	re := regexp.MustCompile(`<!--.*?-->`)
	return re.ReplaceAllString(content, "")
}

// removeCSSComments removes CSS comments
func (r *Remover) removeCSSComments(content string) string {
	re := regexp.MustCompile(`/\*.*?\*/`)
	return re.ReplaceAllString(content, "")
}

// removeSQLComments removes SQL comments
func (r *Remover) removeSQLComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string
	inMultiComment := false

	for _, line := range lines {
		if inMultiComment {
			if idx := strings.Index(line, "*/"); idx != -1 {
				inMultiComment = false
				line = line[idx+2:]
			} else {
				continue
			}
		}

		// Remove -- comments
		if idx := strings.Index(line, "--"); idx != -1 {
			line = line[:idx]
		}

		// Check for /* comments
		if idx := strings.Index(line, "/*"); idx != -1 {
			before := line[:idx]
			after := line[idx+2:]
			if endIdx := strings.Index(after, "*/"); endIdx != -1 {
				line = before + after[endIdx+2:]
			} else {
				inMultiComment = true
				line = before
			}
		}

		result = append(result, strings.TrimRight(line, " \t"))
	}

	return strings.Join(result, "\n")
}
