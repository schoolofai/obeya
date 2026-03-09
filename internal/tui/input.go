package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel handles free-text input from the user using bubbles textinput.
type InputModel struct {
	prompt  string
	context string
	input   textinput.Model
}

func newInputModel(prompt string, context ...string) InputModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 120
	ti.Width = 40
	m := InputModel{
		prompt: prompt,
		input:  ti,
	}
	if len(context) > 0 {
		m.context = context[0]
	}
	return m
}

func (m InputModel) View() string {
	var b strings.Builder
	b.WriteString(m.prompt)
	if m.context != "" {
		b.WriteString(" (" + m.context + ")")
	}
	b.WriteString("\n\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Enter:confirm  Esc:cancel"))
	return overlayStyle.Render(b.String())
}

func (m InputModel) Value() string {
	return m.input.Value()
}

// Update forwards key events to the underlying textinput model.
func (m InputModel) Update(msg tea.KeyMsg) (InputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
