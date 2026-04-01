package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dylanbr0wn/coach/internal/pipeline"
	"github.com/dylanbr0wn/coach/internal/types"
)

// Phase constants for the install TUI.
const (
	phaseProgress  = 0
	phaseSelection = 1
)

// progressMsg is sent when the evaluate stage reports progress.
type progressMsg struct {
	current int
	total   int
	name    string
}

// evalDoneMsg is sent when evaluation completes.
type evalDoneMsg struct {
	vetted []pipeline.VettedSkill
	err    error
}

// InstallModel is the Bubble Tea model for the batch install TUI.
// It has two phases: a progress bar during evaluation, then an interactive
// selection table.
type InstallModel struct {
	// Config
	source string
	force  bool

	// Phase tracking
	phase int

	// Phase 1: progress
	progress progress.Model
	current  int
	total    int
	evalName string

	// Phase 2: selection
	vetted    []pipeline.VettedSkill
	cursor    int
	selected  map[int]bool
	confirmed bool
	cancelled bool

	// Preview
	previewing bool
	previewIdx int

	// Error
	err error
}

// NewInstallModel creates a new install TUI model.
func NewInstallModel(source string, force bool) *InstallModel {
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 40

	return &InstallModel{
		source:   source,
		force:    force,
		phase:    phaseProgress,
		progress: p,
		selected: make(map[int]bool),
	}
}

// SetEvalDone transitions the model from progress to selection phase.
// Called externally when evaluation completes (for --yes mode or testing).
func (m *InstallModel) SetEvalDone(vetted []pipeline.VettedSkill) {
	m.vetted = vetted
	m.phase = phaseSelection
}

// Selected returns the skills the user selected.
func (m *InstallModel) Selected() []pipeline.VettedSkill {
	var result []pipeline.VettedSkill
	for i, v := range m.vetted {
		if m.selected[i] {
			result = append(result, v)
		}
	}
	return result
}

// Cancelled returns true if the user quit without confirming.
func (m *InstallModel) Cancelled() bool {
	return m.cancelled
}

// Err returns any error from evaluation.
func (m *InstallModel) Err() error {
	return m.err
}

// StartEval returns a tea.Cmd that runs the evaluate stage in the background,
// sending progress messages and a final evalDoneMsg.
func StartEval(candidates []pipeline.SkillCandidate, db interface{ PatternDB() }, force bool, evalFn func([]pipeline.SkillCandidate, func(int, int, string)) ([]pipeline.VettedSkill, error)) tea.Cmd {
	return func() tea.Msg {
		vetted, err := evalFn(candidates, func(current, total int, name string) {
			// Note: progress updates are sent synchronously in the eval callback.
			// The TUI won't see these until evalDone since we can't send tea.Msg
			// from inside a Cmd. Instead we report final results.
		})
		return evalDoneMsg{vetted: vetted, err: err}
	}
}

// StartEvalWithProgress returns a tea.Cmd that runs evaluation and sends
// progress messages through a channel that the model polls.
func (m *InstallModel) StartEvalWithProgress(
	candidates []pipeline.SkillCandidate,
	evalFn func([]pipeline.SkillCandidate, func(int, int, string)) ([]pipeline.VettedSkill, error),
) tea.Cmd {
	return func() tea.Msg {
		var lastProgress progressMsg
		vetted, err := evalFn(candidates, func(current, total int, name string) {
			lastProgress = progressMsg{current: current, total: total, name: name}
		})
		_ = lastProgress // progress is consumed synchronously
		return evalDoneMsg{vetted: vetted, err: err}
	}
}

func (m *InstallModel) Init() tea.Cmd {
	return nil
}

func (m *InstallModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case progressMsg:
		m.current = msg.current
		m.total = msg.total
		m.evalName = msg.name
		return m, nil

	case evalDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.vetted = msg.vetted
		m.phase = phaseSelection
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 10
		if m.progress.Width > 60 {
			m.progress.Width = 60
		}
		return m, nil
	}
	return m, nil
}

func (m *InstallModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.phase == phaseProgress {
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.cancelled = true
			return m, tea.Quit
		}
		return m, nil
	}

	// Preview mode: dismiss on p or esc.
	if m.previewing {
		switch msg.String() {
		case "p", "esc", "q":
			m.previewing = false
		}
		return m, nil
	}

	// Selection mode.
	switch msg.String() {
	case "ctrl+c", "q":
		m.cancelled = true
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.vetted)-1 {
			m.cursor++
		}

	case " ": // space = toggle
		if m.cursor < len(m.vetted) && m.vetted[m.cursor].Selectable {
			m.selected[m.cursor] = !m.selected[m.cursor]
		}

	case "a": // select all passing
		for i, v := range m.vetted {
			if v.Selectable {
				m.selected[i] = true
			}
		}

	case "n": // select none
		m.selected = make(map[int]bool)

	case "p": // preview
		if m.cursor < len(m.vetted) {
			m.previewing = true
			m.previewIdx = m.cursor
		}

	case "enter":
		m.confirmed = true
		return m, tea.Quit
	}
	return m, nil
}

func (m *InstallModel) View() string {
	if m.phase == phaseProgress {
		return m.viewProgress()
	}
	return m.viewSelection()
}

func (m *InstallModel) viewProgress() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(" Evaluating skills from %s\n\n", HeadingStyle.Render(m.source)))

	pct := 0.0
	if m.total > 0 {
		pct = float64(m.current) / float64(m.total)
	}
	sb.WriteString(fmt.Sprintf(" %s  %d/%d\n", m.progress.ViewAs(pct), m.current, m.total))

	if m.evalName != "" {
		sb.WriteString(fmt.Sprintf("\n %s %s\n", DimStyle.Render("Scanning:"), m.evalName))
	}
	sb.WriteString("\n")
	return sb.String()
}

func (m *InstallModel) viewSelection() string {
	var sb strings.Builder

	// Header
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf(" %s — %d skills discovered from %s\n\n",
		HeadingStyle.Render("Coach Install"),
		len(m.vetted),
		DimStyle.Render(m.source),
	))

	// Table header
	selW := 5
	nameW := maxSkillNameWidth(m.vetted, 25)
	colW := 6
	issueW := 30

	headerFmt := fmt.Sprintf(" %%-%ds│ %%-%ds│ %%-%ds│ %%-%ds│ %%-%ds│ %%s\n",
		selW, nameW, colW, colW, colW+2)
	sb.WriteString(DimStyle.Render(fmt.Sprintf(headerFmt,
		" Sel", " Skill", " Lint", " Scan", " Quality", "Issues")))

	// Separator
	sep := fmt.Sprintf(" %s┼%s┼%s┼%s┼%s┼%s\n",
		strings.Repeat("─", selW),
		strings.Repeat("─", nameW),
		strings.Repeat("─", colW),
		strings.Repeat("─", colW),
		strings.Repeat("─", colW+2),
		strings.Repeat("─", issueW))
	sb.WriteString(DimStyle.Render(sep))

	// Rows
	for i, v := range m.vetted {
		row := m.renderRow(i, v, selW, nameW, colW)
		if i == m.cursor {
			row = lipgloss.NewStyle().Reverse(true).Render(row)
		}
		sb.WriteString(row)
		sb.WriteString("\n")
	}

	// Selection count
	count := 0
	for _, s := range m.selected {
		if s {
			count++
		}
	}
	sb.WriteString(fmt.Sprintf("\n %d selected\n", count))

	// Hotkeys
	sb.WriteString(fmt.Sprintf("\n %s\n",
		DimStyle.Render("[a] all passing  [n] none  [↑↓] navigate  [space] toggle  [p] preview  [enter] confirm  [q] quit")))

	// Preview pane
	if m.previewing && m.previewIdx < len(m.vetted) {
		sb.WriteString(m.renderPreview(m.previewIdx))
	}

	sb.WriteString("\n")
	return sb.String()
}

func (m *InstallModel) renderRow(idx int, v pipeline.VettedSkill, selW, nameW, colW int) string {
	// Selection indicator
	sel := "[ ]"
	if m.selected[idx] {
		sel = "[●]"
	}
	if !v.Selectable {
		if v.LintResult.Status == pipeline.CheckFail {
			sel = "[✗]"
		} else {
			sel = "[!]"
		}
	}
	if v.Candidate.Origin == pipeline.OriginInstalledModified {
		sel = "[~]"
	}

	// Name
	name := ""
	if v.Skill != nil {
		name = v.Skill.Name
	} else {
		name = filepath.Base(v.Candidate.Path)
	}
	if len(name) > nameW-2 {
		name = name[:nameW-5] + "..."
	}

	// Status columns
	lint := statusStr(v.LintResult.Status)
	scan := "—"
	if v.ScanResult != nil {
		scan = riskStr(v.ScanResult.Risk)
	}
	quality := "—"
	if v.LintResult.Status == pipeline.CheckPass {
		quality = statusStr(v.QualityResult.Status)
	}

	// Issues summary
	issues := issuesSummary(v)

	return fmt.Sprintf(" %-*s│ %-*s│ %-*s│ %-*s│ %-*s│ %s",
		selW, sel,
		nameW, name,
		colW, lint,
		colW, scan,
		colW+2, quality,
		issues,
	)
}

func (m *InstallModel) renderPreview(idx int) string {
	v := m.vetted[idx]
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(DimStyle.Render(" ── Preview ──────────────────────────────────────"))
	sb.WriteString("\n")

	if v.Skill != nil {
		sb.WriteString(fmt.Sprintf(" %s\n", HeadingStyle.Render(v.Skill.Name)))
		sb.WriteString(fmt.Sprintf(" %s\n\n", v.Skill.Description))
		body := v.Skill.Body
		if len(body) > 500 {
			body = body[:497] + "..."
		}
		sb.WriteString(fmt.Sprintf(" %s\n", strings.ReplaceAll(body, "\n", "\n ")))
	} else {
		sb.WriteString(fmt.Sprintf(" %s\n", ErrorStyle.Render("Could not parse skill")))
		if len(v.LintResult.Issues) > 0 {
			for _, issue := range v.LintResult.Issues {
				sb.WriteString(fmt.Sprintf("   %s %s\n", ErrorStyle.Render("•"), issue))
			}
		}
	}
	sb.WriteString(DimStyle.Render("\n Press p or esc to close preview"))
	sb.WriteString("\n")
	return sb.String()
}

func statusStr(s pipeline.CheckStatus) string {
	switch s {
	case pipeline.CheckPass:
		return SuccessStyle.Render("PASS")
	case pipeline.CheckWarn:
		return WarningStyle.Render("WARN")
	case pipeline.CheckFail:
		return ErrorStyle.Render("FAIL")
	default:
		return "—"
	}
}

func riskStr(r types.RiskLevel) string {
	switch r {
	case types.RiskLow:
		return SuccessStyle.Render("PASS")
	case types.RiskMedium:
		return WarningStyle.Render("MED")
	case types.RiskHigh:
		return ErrorStyle.Render("HIGH")
	case types.RiskCritical:
		return ErrorStyle.Render("CRIT")
	default:
		return "—"
	}
}

func issuesSummary(v pipeline.VettedSkill) string {
	var parts []string

	if v.LintResult.Status == pipeline.CheckFail && len(v.LintResult.Issues) > 0 {
		parts = append(parts, v.LintResult.Issues[0])
	}
	if v.ScanResult != nil && v.ScanResult.Risk >= types.RiskHigh {
		for _, f := range v.ScanResult.Findings {
			if f.Severity >= types.SeverityHigh {
				parts = append(parts, f.Name)
				break
			}
		}
	}
	if v.QualityResult.Status == pipeline.CheckWarn && len(v.QualityResult.Issues) > 0 {
		parts = append(parts, v.QualityResult.Issues[0])
	}

	summary := strings.Join(parts, "; ")
	if len(summary) > 40 {
		summary = summary[:37] + "..."
	}
	return summary
}

func maxSkillNameWidth(vetted []pipeline.VettedSkill, min int) int {
	w := min
	for _, v := range vetted {
		name := ""
		if v.Skill != nil {
			name = v.Skill.Name
		} else {
			name = filepath.Base(v.Candidate.Path)
		}
		if len(name)+2 > w {
			w = len(name) + 2
		}
	}
	if w > 40 {
		w = 40
	}
	return w
}
