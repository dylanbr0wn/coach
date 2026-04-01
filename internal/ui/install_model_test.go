package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dylanbr0wn/coach/internal/pipeline"
	"github.com/dylanbr0wn/coach/internal/types"
)

func makeVetted(name string, selectable bool, lintStatus pipeline.CheckStatus, risk types.RiskLevel) pipeline.VettedSkill {
	v := pipeline.VettedSkill{
		Candidate: pipeline.SkillCandidate{
			Path:   "/tmp/" + name,
			Source: "test",
			SHA:    "abc",
			Origin: pipeline.OriginLocal,
		},
		Selectable: selectable,
		LintResult: pipeline.CheckResult{Status: lintStatus},
	}
	if lintStatus == pipeline.CheckPass {
		v.Skill = &types.Skill{
			Name:        name,
			Description: "A test skill for " + name,
			Body:        "# " + name + "\n\nSome instructions here.",
		}
		v.ScanResult = &types.ScanResult{Score: int(risk) * 25, Risk: risk}
		v.QualityResult = pipeline.CheckResult{Status: pipeline.CheckPass}
	}
	return v
}

func sendKey(m *InstallModel, key string) {
	km := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case "up":
		km = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		km = tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		km = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		km = tea.KeyMsg{Type: tea.KeyEscape}
	case "ctrl+c":
		km = tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	m.Update(km)
}

func TestInstallModel_InitialState(t *testing.T) {
	m := NewInstallModel("test-source", false)
	if m.phase != phaseProgress {
		t.Errorf("initial phase = %d, want %d (progress)", m.phase, phaseProgress)
	}
	if m.Cancelled() {
		t.Error("should not be cancelled initially")
	}
}

func TestInstallModel_EvalDoneTransitionsToSelection(t *testing.T) {
	m := NewInstallModel("test-source", false)
	vetted := []pipeline.VettedSkill{
		makeVetted("skill-a", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("skill-b", true, pipeline.CheckPass, types.RiskLow),
	}

	m.Update(evalDoneMsg{vetted: vetted})
	if m.phase != phaseSelection {
		t.Errorf("phase after evalDone = %d, want %d (selection)", m.phase, phaseSelection)
	}
	if len(m.vetted) != 2 {
		t.Errorf("vetted count = %d, want 2", len(m.vetted))
	}
}

func TestInstallModel_EvalDoneError(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.Update(evalDoneMsg{err: fmt.Errorf("eval failed")})
	if m.Err() == nil {
		t.Error("expected error after evalDoneMsg with err")
	}
}

func TestInstallModel_SelectAllPassing(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("good-1", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("good-2", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("bad-1", false, pipeline.CheckFail, types.RiskLow),
	})

	sendKey(m, "a")
	selected := m.Selected()
	if len(selected) != 2 {
		t.Errorf("selected %d skills, want 2 (only passing)", len(selected))
	}
}

func TestInstallModel_SelectNone(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("good-1", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("good-2", true, pipeline.CheckPass, types.RiskLow),
	})

	sendKey(m, "a") // select all
	sendKey(m, "n") // then none
	if len(m.Selected()) != 0 {
		t.Errorf("selected %d skills after 'n', want 0", len(m.Selected()))
	}
}

func TestInstallModel_ToggleSpace(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("skill-a", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("skill-b", true, pipeline.CheckPass, types.RiskLow),
	})

	// Cursor starts at 0, toggle skill-a on
	sendKey(m, " ")
	if !m.selected[0] {
		t.Error("skill-a should be selected after space")
	}

	// Toggle off
	sendKey(m, " ")
	if m.selected[0] {
		t.Error("skill-a should be deselected after second space")
	}
}

func TestInstallModel_SpaceOnUnselectable(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("bad", false, pipeline.CheckFail, types.RiskLow),
	})

	sendKey(m, " ")
	if m.selected[0] {
		t.Error("unselectable skill should not be toggled")
	}
}

func TestInstallModel_Navigation(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("a", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("b", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("c", true, pipeline.CheckPass, types.RiskLow),
	})

	if m.cursor != 0 {
		t.Fatalf("cursor starts at %d, want 0", m.cursor)
	}

	sendKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", m.cursor)
	}

	sendKey(m, "j")
	if m.cursor != 2 {
		t.Errorf("after j: cursor = %d, want 2", m.cursor)
	}

	// Can't go past end
	sendKey(m, "down")
	if m.cursor != 2 {
		t.Errorf("past end: cursor = %d, want 2", m.cursor)
	}

	sendKey(m, "up")
	if m.cursor != 1 {
		t.Errorf("after up: cursor = %d, want 1", m.cursor)
	}

	sendKey(m, "k")
	if m.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", m.cursor)
	}

	// Can't go before start
	sendKey(m, "up")
	if m.cursor != 0 {
		t.Errorf("before start: cursor = %d, want 0", m.cursor)
	}
}

func TestInstallModel_QuitCancels(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("a", true, pipeline.CheckPass, types.RiskLow),
	})

	sendKey(m, "q")
	if !m.Cancelled() {
		t.Error("q should cancel")
	}
}

func TestInstallModel_EnterConfirms(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("a", true, pipeline.CheckPass, types.RiskLow),
	})

	sendKey(m, " ") // select
	sendKey(m, "enter")
	if !m.confirmed {
		t.Error("enter should confirm")
	}
	if len(m.Selected()) != 1 {
		t.Errorf("selected = %d, want 1", len(m.Selected()))
	}
}

func TestInstallModel_Preview(t *testing.T) {
	m := NewInstallModel("test-source", false)
	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("a", true, pipeline.CheckPass, types.RiskLow),
	})

	sendKey(m, "p")
	if !m.previewing {
		t.Error("p should open preview")
	}

	// In preview mode, other keys should not navigate
	sendKey(m, "esc")
	if m.previewing {
		t.Error("esc should close preview")
	}
}

func TestInstallModel_ViewRenders(t *testing.T) {
	m := NewInstallModel("test-source", false)

	// Progress phase should render without panic
	view := m.View()
	if view == "" {
		t.Error("progress view should not be empty")
	}

	m.SetEvalDone([]pipeline.VettedSkill{
		makeVetted("good", true, pipeline.CheckPass, types.RiskLow),
		makeVetted("bad", false, pipeline.CheckFail, types.RiskLow),
	})

	// Selection phase should render without panic
	view = m.View()
	if view == "" {
		t.Error("selection view should not be empty")
	}
	if !containsStr(view, "Coach Install") {
		t.Error("selection view should contain header")
	}
	if !containsStr(view, "good") {
		t.Error("selection view should contain skill name")
	}
}

func TestInstallModel_SetEvalDone(t *testing.T) {
	m := NewInstallModel("test-source", false)
	vetted := []pipeline.VettedSkill{
		makeVetted("a", true, pipeline.CheckPass, types.RiskLow),
	}
	m.SetEvalDone(vetted)

	if m.phase != phaseSelection {
		t.Errorf("phase = %d, want selection", m.phase)
	}
	if len(m.vetted) != 1 {
		t.Errorf("vetted = %d, want 1", len(m.vetted))
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
