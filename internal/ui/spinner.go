package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type errMsg struct{ err error }

type spinnerModel struct {
	spinner spinner.Model
	msg     string
	fn      func() error
	done    bool
	err     error
}

func (m *spinnerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		return errMsg{err: m.fn()}
	})
}

func (m *spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg.err
		m.done = true
		return m, tea.Quit
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.done = true
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *spinnerModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("  %s %s\n", m.spinner.View(), m.msg)
}

// WithSpinner runs fn while displaying an animated spinner with the given message.
// The spinner renders to stderr so it doesn't interfere with piped output.
// If bubbletea fails to initialize, fn is run without a spinner as a fallback.
// Note: fn runs exactly once — either inside bubbletea or in the fallback path.
func WithSpinner(msg string, fn func() error) error {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = InfoStyle

	m := &spinnerModel{
		spinner: s,
		msg:     msg,
		fn:      fn,
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		// Bubbletea failed to start — fn may not have run yet.
		// Check if it already executed inside the program.
		if fm, ok := finalModel.(*spinnerModel); ok && fm.done {
			return fm.err
		}
		// fn never ran — execute it without spinner.
		return fn()
	}

	return finalModel.(*spinnerModel).err
}
