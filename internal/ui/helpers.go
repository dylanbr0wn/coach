package ui

import "fmt"

// Success returns a formatted success message: ✓ msg
func Success(msg string) string {
	return fmt.Sprintf("%s %s", SuccessStyle.Render("✓"), msg)
}

// Warn returns a formatted warning message with an optional hint.
// When hint is non-empty, a second line with a → arrow is appended.
func Warn(msg, hint string) string {
	line := fmt.Sprintf("%s %s", WarningStyle.Render("⚠"), msg)
	if hint != "" {
		line += fmt.Sprintf("\n  %s %s", InfoStyle.Render("→"), hint)
	}
	return line
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
