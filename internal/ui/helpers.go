package ui

import "fmt"

// Success returns a formatted success message: ✓ msg
func Success(msg string) string {
	return fmt.Sprintf("%s %s", SuccessStyle.Render("✓"), msg)
}

// Warn returns a formatted warning message: ⚠ msg
func Warn(msg string) string {
	return fmt.Sprintf("%s %s", WarningStyle.Render("⚠"), msg)
}

// Error returns a formatted error message with an optional suggestion.
func Error(msg, suggestion string) string {
	line := fmt.Sprintf("%s %s", ErrorStyle.Render("✗"), msg)
	if suggestion != "" {
		line += fmt.Sprintf("\n  %s %s", InfoStyle.Render("→"), suggestion)
	}
	return line
}

// NextStep returns a dimmed hint for what command to run next.
func NextStep(cmd, desc string) string {
	return DimStyle.Render(fmt.Sprintf("Next: coach %s — %s", cmd, desc))
}
