package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dylanbr0wn/coach/internal/types"
)

func RenderFindings(findings []types.Finding) string {
	if len(findings) == 0 {
		return SuccessStyle.Render("  No issues found")
	}
	var sb strings.Builder
	for _, f := range findings {
		icon := severityIcon(f.Severity)
		style := severityStyle(f.Severity)
		tag := DimStyle.Render("[" + f.Category + "]")
		sb.WriteString(fmt.Sprintf("  %s %s %s\n", icon, tag, style.Render(f.Name)))
		sb.WriteString(fmt.Sprintf("    %s %s\n", DimStyle.Render(f.ID), f.Description))
		if f.File != "" {
			location := f.File
			if f.Line > 0 {
				location = fmt.Sprintf("%s:%d", f.File, f.Line)
			}
			sb.WriteString(fmt.Sprintf("    %s %s\n", DimStyle.Render("at"), location))
		}
		if f.Match != "" {
			truncated := f.Match
			if len(truncated) > 60 {
				truncated = truncated[:57] + "..."
			}
			sb.WriteString(fmt.Sprintf("    %s %s\n", DimStyle.Render("match:"), truncated))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func RenderScanSummary(result *types.ScanResult) string {
	riskStyle := riskLevelStyle(result.Risk)
	skillName := result.SkillPath
	header := fmt.Sprintf("  %s — Risk: %s (score: %d/100)",
		HeadingStyle.Render(skillName),
		riskStyle.Render(result.Risk.String()),
		result.Score,
	)
	counts := countBySeverity(result.Findings)
	var parts []string
	if counts[types.SeverityCritical] > 0 {
		parts = append(parts, ErrorStyle.Render(fmt.Sprintf("%d critical", counts[types.SeverityCritical])))
	}
	if counts[types.SeverityHigh] > 0 {
		parts = append(parts, ErrorStyle.Render(fmt.Sprintf("%d high", counts[types.SeverityHigh])))
	}
	if counts[types.SeverityWarning] > 0 {
		parts = append(parts, WarningStyle.Render(fmt.Sprintf("%d warnings", counts[types.SeverityWarning])))
	}
	summary := ""
	if len(parts) > 0 {
		summary = "\n  " + strings.Join(parts, ", ")
	}
	return BoxStyle.Render(header + summary)
}

func severityIcon(s types.Severity) string {
	switch {
	case s >= types.SeverityCritical:
		return ErrorStyle.Render("✗")
	case s >= types.SeverityHigh:
		return ErrorStyle.Render("!")
	case s >= types.SeverityWarning:
		return WarningStyle.Render("⚠")
	default:
		return InfoStyle.Render("ℹ")
	}
}

func severityStyle(s types.Severity) lipgloss.Style {
	switch {
	case s >= types.SeverityHigh:
		return ErrorStyle
	case s >= types.SeverityWarning:
		return WarningStyle
	default:
		return InfoStyle
	}
}

func riskLevelStyle(r types.RiskLevel) lipgloss.Style {
	switch r {
	case types.RiskCritical, types.RiskHigh:
		return ErrorStyle
	case types.RiskMedium:
		return WarningStyle
	default:
		return SuccessStyle
	}
}

func countBySeverity(findings []types.Finding) map[types.Severity]int {
	counts := make(map[types.Severity]int)
	for _, f := range findings {
		counts[f.Severity]++
	}
	return counts
}
