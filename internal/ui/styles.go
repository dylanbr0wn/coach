package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

// PromptYesNo prints prompt to stdout and reads a yes/no answer from stdin.
// Returns true for empty input, "y", or "yes" (case-insensitive).
func PromptYesNo(prompt string) bool {
	fmt.Print(prompt)
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return false
	}
	answer := strings.TrimSpace(strings.ToLower(sc.Text()))
	return answer == "" || answer == "y" || answer == "yes"
}
