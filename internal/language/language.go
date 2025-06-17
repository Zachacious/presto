package language

import (
	"path/filepath"
	"strings"

	"github.com/Zachacious/presto/pkg/types"
)

var extensionMap = map[string]types.Language{
	".go":   types.LangGo,
	".js":   types.LangJavaScript,
	".mjs":  types.LangJavaScript,
	".jsx":  types.LangJavaScript,
	".ts":   types.LangTypeScript,
	".tsx":  types.LangTypeScript,
	".py":   types.LangPython,
	".java": types.LangJava,
	".c":    types.LangC,
	".h":    types.LangC,
	".cpp":  types.LangCPP,
	".cxx":  types.LangCPP,
	".cc":   types.LangCPP,
	".hpp":  types.LangCPP,
	".rs":   types.LangRust,
	".html": types.LangHTML,
	".htm":  types.LangHTML,
	".css":  types.LangCSS,
	".sql":  types.LangSQL,
	".json": types.LangJSON,
	".yaml": types.LangYAML,
	".yml":  types.LangYAML,
	".md":   types.LangMarkdown,
	".sh":   types.LangShell,
	".bash": types.LangShell,
	".zsh":  types.LangShell,
	".fish": types.LangShell,
	".txt":  types.LangText,
}

var commentStyles = map[types.Language]types.CommentStyle{
	types.LangGo: {
		SingleLine: "//",
		MultiStart: "/*",
		MultiEnd:   "*/",
		DocStyle:   []string{"//"},
	},
	types.LangJavaScript: {
		SingleLine: "//",
		MultiStart: "/*",
		MultiEnd:   "*/",
		DocStyle:   []string{"/**"},
	},
	types.LangTypeScript: {
		SingleLine: "//",
		MultiStart: "/*",
		MultiEnd:   "*/",
		DocStyle:   []string{"/**"},
	},
	types.LangPython: {
		SingleLine: "#",
		MultiStart: `"""`,
		MultiEnd:   `"""`,
		DocStyle:   []string{`"""`},
	},
	types.LangJava: {
		SingleLine: "//",
		MultiStart: "/*",
		MultiEnd:   "*/",
		DocStyle:   []string{"/**"},
	},
	types.LangC: {
		SingleLine: "//",
		MultiStart: "/*",
		MultiEnd:   "*/",
		DocStyle:   []string{"/**"},
	},
	types.LangCPP: {
		SingleLine: "//",
		MultiStart: "/*",
		MultiEnd:   "*/",
		DocStyle:   []string{"/**"},
	},
	types.LangRust: {
		SingleLine: "//",
		MultiStart: "/*",
		MultiEnd:   "*/",
		DocStyle:   []string{"///", "//!"},
	},
	types.LangHTML: {
		MultiStart: "<!--",
		MultiEnd:   "-->",
	},
	types.LangCSS: {
		MultiStart: "/*",
		MultiEnd:   "*/",
	},
	types.LangSQL: {
		SingleLine: "--",
		MultiStart: "/*",
		MultiEnd:   "*/",
	},
	types.LangShell: {
		SingleLine: "#",
	},
	types.LangYAML: {
		SingleLine: "#",
	},
}

// DetectLanguage detects programming language from file extension
func DetectLanguage(filename string) types.Language {
	ext := strings.ToLower(filepath.Ext(filename))
	if lang, exists := extensionMap[ext]; exists {
		return lang
	}
	return types.LangUnknown
}

// GetCommentStyle returns the comment style for a language
func GetCommentStyle(lang types.Language) types.CommentStyle {
	if style, exists := commentStyles[lang]; exists {
		return style
	}
	return types.CommentStyle{} // No comment style
}

// IsTextFile determines if a file should be processed as text
func IsTextFile(lang types.Language) bool {
	return lang != types.LangUnknown
}
