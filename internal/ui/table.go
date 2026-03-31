package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TableRow struct {
	Cells []string
}

func RenderTable(headers []string, rows []TableRow) string {
	if len(rows) == 0 {
		return DimStyle.Render("  No data to display")
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row.Cells {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	for i := range widths {
		widths[i] += 2
	}
	var sb strings.Builder
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(White)
	var headerParts []string
	for i, h := range headers {
		headerParts = append(headerParts, headerStyle.Width(widths[i]).Render(h))
	}
	sb.WriteString("  " + strings.Join(headerParts, "") + "\n")
	var sepParts []string
	for _, w := range widths {
		sepParts = append(sepParts, DimStyle.Render(strings.Repeat("─", w)))
	}
	sb.WriteString("  " + strings.Join(sepParts, "") + "\n")
	for _, row := range rows {
		var cellParts []string
		for i, cell := range row.Cells {
			w := widths[0]
			if i < len(widths) {
				w = widths[i]
			}
			style := lipgloss.NewStyle().Width(w)
			cellParts = append(cellParts, style.Render(cell))
		}
		sb.WriteString("  " + strings.Join(cellParts, "") + "\n")
	}
	return sb.String()
}

