package language

import (
	"path/filepath"
	"strings"

	"github.com/Zachacious/presto/pkg/types"
)

// DetectLanguage detects the programming language from file extension
func DetectLanguage(filePath string) types.Language {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".go":
		return types.LangGo
	case ".js", ".jsx":
		return types.LangJavaScript
	case ".ts", ".tsx":
		return types.LangTypeScript
	case ".py":
		return types.LangPython
	case ".java":
		return types.LangJava
	case ".c":
		return types.LangC
	case ".cpp", ".cc", ".cxx":
		return types.LangCPP
	case ".rs":
		return types.LangRust
	case ".php":
		return types.LangPHP
	case ".rb":
		return types.LangRuby
	case ".sh", ".bash", ".zsh":
		return types.LangShell
	case ".sql":
		return types.LangSQL
	case ".html", ".htm":
		return types.LangHTML
	case ".css":
		return types.LangCSS
	case ".xml":
		return types.LangXML
	case ".json":
		return types.LangJSON
	case ".yaml", ".yml":
		return types.LangYAML
	case ".md", ".markdown":
		return types.LangMarkdown
	case ".txt", ".text", "":
		return types.LangText
	default:
		return types.LangUnknown
	}
}

// IsTextFile determines if a language represents a text file that can be processed
func IsTextFile(lang types.Language) bool {
	switch lang {
	case types.LangUnknown:
		return false
	default:
		return true
	}
}

// GetCommentStyle returns the comment style for a language
func GetCommentStyle(lang types.Language) types.CommentStyle {
	switch lang {
	case types.LangGo, types.LangJava, types.LangJavaScript, types.LangTypeScript,
		types.LangC, types.LangCPP, types.LangRust, types.LangPHP:
		return types.CommentStyle{
			LineComment: "//",
			BlockStart:  "/*",
			BlockEnd:    "*/",
		}
	case types.LangPython, types.LangShell, types.LangRuby:
		return types.CommentStyle{
			LineComment: "#",
		}
	case types.LangSQL:
		return types.CommentStyle{
			LineComment: "--",
			BlockStart:  "/*",
			BlockEnd:    "*/",
		}
	case types.LangHTML, types.LangXML:
		return types.CommentStyle{
			BlockStart: "<!--",
			BlockEnd:   "-->",
		}
	case types.LangCSS:
		return types.CommentStyle{
			BlockStart: "/*",
			BlockEnd:   "*/",
		}
	default:
		return types.CommentStyle{}
	}
}

// GetFileExtensions returns common file extensions for a language
func GetFileExtensions(lang types.Language) []string {
	switch lang {
	case types.LangGo:
		return []string{".go"}
	case types.LangJavaScript:
		return []string{".js", ".jsx"}
	case types.LangTypeScript:
		return []string{".ts", ".tsx"}
	case types.LangPython:
		return []string{".py"}
	case types.LangJava:
		return []string{".java"}
	case types.LangC:
		return []string{".c", ".h"}
	case types.LangCPP:
		return []string{".cpp", ".cc", ".cxx", ".hpp"}
	case types.LangRust:
		return []string{".rs"}
	case types.LangPHP:
		return []string{".php"}
	case types.LangRuby:
		return []string{".rb"}
	case types.LangShell:
		return []string{".sh", ".bash", ".zsh"}
	case types.LangSQL:
		return []string{".sql"}
	case types.LangHTML:
		return []string{".html", ".htm"}
	case types.LangCSS:
		return []string{".css"}
	case types.LangXML:
		return []string{".xml"}
	case types.LangJSON:
		return []string{".json"}
	case types.LangYAML:
		return []string{".yaml", ".yml"}
	case types.LangMarkdown:
		return []string{".md", ".markdown"}
	case types.LangText:
		return []string{".txt", ".text"}
	default:
		return []string{}
	}
}
