package tui

import (
	"path/filepath"
	"strings"
)

func colorDiffLine(line, path string) string {
	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return codeMuted.Render(line)
	case strings.HasPrefix(line, "+"):
		return addStyle.Render("+") + highlightCode(visualText(line[1:]), path)
	case strings.HasPrefix(line, "-"):
		return delStyle.Render("-") + highlightCode(visualText(line[1:]), path)
	case strings.HasPrefix(line, " "):
		return " " + highlightCode(visualText(line[1:]), path)
	case strings.HasPrefix(line, "@@"):
		return hunkStyle.Render(line)
	default:
		return highlightCode(visualText(line), path)
	}
}
func highlightCode(line, path string) string {
	if !highlightable(path) || strings.TrimSpace(line) == "" {
		return line
	}

	comment := ""
	if match := commentPattern.FindStringIndex(line); match != nil {
		comment = commentStyle.Render(line[match[0]:])
		line = line[:match[0]]
	}

	line = stringPattern.ReplaceAllStringFunc(line, func(match string) string {
		return stringStyle.Render(match)
	})
	line = tagPattern.ReplaceAllStringFunc(line, func(match string) string {
		return tagStyle.Render(match)
	})
	line = keywordPattern.ReplaceAllStringFunc(line, func(match string) string {
		return keywordStyle.Render(match)
	})

	return line + comment
}
func highlightable(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".json", ".md", ".css", ".scss", ".html", ".vue", ".svelte", ".py", ".rb", ".rs", ".java", ".kt", ".swift", ".php", ".sh", ".yml", ".yaml", ".toml":
		return true
	default:
		return false
	}
}
