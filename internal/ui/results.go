package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/dylan/coach/pkg"
)

func RenderFindings(findings []pkg.Finding) string {
	if len(findings) == 0 {
		return SuccessStyle.Render("  No issues found")
	}
	var sb strings.Builder
	for _, f := range findings {
		icon := severityIcon(f.Severity)
		style := severityStyle(f.Severity)
		sb.WriteString(fmt.Sprintf("  %s %s\n", icon, style.Render(f.Name)))
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

func RenderScanSummary(result *pkg.ScanResult) string {
	riskStyle := riskLevelStyle(result.Risk)
	skillName := result.SkillPath
	header := fmt.Sprintf("  %s — Risk: %s (score: %d/100)",
		HeadingStyle.Render(skillName),
		riskStyle.Render(result.Risk.String()),
		result.Score,
	)
	counts := countBySeverity(result.Findings)
	var parts []string
	if counts[pkg.SeverityCritical] > 0 {
		parts = append(parts, ErrorStyle.Render(fmt.Sprintf("%d critical", counts[pkg.SeverityCritical])))
	}
	if counts[pkg.SeverityHigh] > 0 {
		parts = append(parts, ErrorStyle.Render(fmt.Sprintf("%d high", counts[pkg.SeverityHigh])))
	}
	if counts[pkg.SeverityWarning] > 0 {
		parts = append(parts, WarningStyle.Render(fmt.Sprintf("%d warnings", counts[pkg.SeverityWarning])))
	}
	summary := ""
	if len(parts) > 0 {
		summary = "\n  " + strings.Join(parts, ", ")
	}
	return BoxStyle.Render(header + summary)
}

func severityIcon(s pkg.Severity) string {
	switch {
	case s >= pkg.SeverityCritical:
		return ErrorStyle.Render("✗")
	case s >= pkg.SeverityHigh:
		return ErrorStyle.Render("!")
	case s >= pkg.SeverityWarning:
		return WarningStyle.Render("⚠")
	default:
		return InfoStyle.Render("ℹ")
	}
}

func severityStyle(s pkg.Severity) lipgloss.Style {
	switch {
	case s >= pkg.SeverityHigh:
		return ErrorStyle
	case s >= pkg.SeverityWarning:
		return WarningStyle
	default:
		return InfoStyle
	}
}

func riskLevelStyle(r pkg.RiskLevel) lipgloss.Style {
	switch r {
	case pkg.RiskCritical, pkg.RiskHigh:
		return ErrorStyle
	case pkg.RiskMedium:
		return WarningStyle
	default:
		return SuccessStyle
	}
}

func countBySeverity(findings []pkg.Finding) map[pkg.Severity]int {
	counts := make(map[pkg.Severity]int)
	for _, f := range findings {
		counts[f.Severity]++
	}
	return counts
}
