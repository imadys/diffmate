package tui

import (
	"github.com/charmbracelet/lipgloss"
	"regexp"
)

var (
	appStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	subtleStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("60")).Bold(true)
	linkedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Background(lipgloss.Color("238"))
	panelStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238"))
	headerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Background(lipgloss.Color("235")).Bold(true)
	keyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	keyBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("248"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("248")).Background(lipgloss.Color("235")).Padding(0, 1)
	addStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	delStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	hunkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	codeMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
	keywordStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	stringStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("150"))
	commentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
	tagStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

	commentPattern = regexp.MustCompile(`(//.*|#.*)$`)
	stringPattern  = regexp.MustCompile(`"[^"]*"|'[^']*'|` + "`[^`]*`")
	tagPattern     = regexp.MustCompile(`</?[A-Za-z][A-Za-z0-9_.:-]*`)
	keywordPattern = regexp.MustCompile(`\b(import|from|export|function|const|let|var|type|interface|return|if|else|for|while|switch|case|default|class|extends|async|await|try|catch|new|nil|null|true|false|package|func|struct|map|range|go|defer|select)\b`)
)
