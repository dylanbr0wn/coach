package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	Green  = lipgloss.Color("#00FF00")
	Yellow = lipgloss.Color("#FFFF00")
	Red    = lipgloss.Color("#FF0000")
	Cyan   = lipgloss.Color("#00FFFF")
	Gray   = lipgloss.Color("#888888")
	White  = lipgloss.Color("#FFFFFF")

	// Styles for findings
	ErrorStyle   = lipgloss.NewStyle().Foreground(Red).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(Yellow)
	InfoStyle    = lipgloss.NewStyle().Foreground(Cyan)
	SuccessStyle = lipgloss.NewStyle().Foreground(Green)
	DimStyle     = lipgloss.NewStyle().Foreground(Gray)

	// Styles for headings / structure
	HeadingStyle = lipgloss.NewStyle().Bold(true).Foreground(White)
	LabelStyle   = lipgloss.NewStyle().Foreground(Gray).Width(14)

	// Box style for status/scan summaries
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Cyan).
			Padding(0, 1)
)
