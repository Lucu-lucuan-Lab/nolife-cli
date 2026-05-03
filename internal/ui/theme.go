package ui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("62")  // Indigo  — Bubbles title bar
	ColorAccent    = lipgloss.Color("170") // Orchid  — Bubbles selected cursor
	ColorBright    = lipgloss.Color("212") // Fuchsia — interactive highlights
	ColorSuccess   = lipgloss.Color("10")  // Green
	ColorError     = lipgloss.Color("9")   // Red
	ColorWarning   = lipgloss.Color("11")  // Yellow
	ColorInfo      = lipgloss.Color("14")  // Cyan
	ColorHighlight = lipgloss.Color("87")  // Bright cyan — filenames, names
	ColorText      = lipgloss.Color("252") // Light gray  — primary text
	ColorSubtext   = lipgloss.Color("243") // Medium gray — labels
	ColorMuted     = lipgloss.Color("240") // Dark gray   — dim / faint
	ColorTitleFg   = lipgloss.Color("230") // Warm white  — title text
)

var (
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorBright)
	BoldStyle  = lipgloss.NewStyle().Bold(true).Foreground(ColorText)
	MutedStyle = lipgloss.NewStyle().Foreground(ColorSubtext)
	FaintStyle = lipgloss.NewStyle().Faint(true)
)

var (
	SuccessStyle = lipgloss.NewStyle().Foreground(ColorSuccess)
	ErrorStyle   = lipgloss.NewStyle().Foreground(ColorError)
	WarningStyle = lipgloss.NewStyle().Foreground(ColorWarning)
	InfoStyle    = lipgloss.NewStyle().Foreground(ColorInfo)
)

var (
	SelectedStyle       = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	SelectedBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorSuccess).
				Padding(0, 1)
)

var (
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 2)
	HeaderTagStyle = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(ColorTitleFg).
			Padding(0, 1)
)

var HelpStyle = lipgloss.NewStyle().Faint(true).PaddingLeft(2)

var (
	PrefixOK   = SuccessStyle.Render("✓")
	PrefixFail = ErrorStyle.Render("✗")
	PrefixWarn = WarningStyle.Render("!")
	PrefixInfo = InfoStyle.Render("●")
)
