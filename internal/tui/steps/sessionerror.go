package steps

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/ssmexec"
	"github.com/eichemberger/burrow/internal/ui"
)

type SessionErrorModel struct {
	failure   ssmexec.RunFailure
	fromSaved bool
}

func NewSessionErrorModel(failure ssmexec.RunFailure, fromSaved bool) *SessionErrorModel {
	return &SessionErrorModel{
		failure:   failure,
		fromSaved: fromSaved,
	}
}

func (m *SessionErrorModel) Init() tea.Cmd {
	return nil
}

func (m *SessionErrorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, func() tea.Msg { return QuitMsg{} }
		case "n", "N":
			return m, func() tea.Msg { return NewConnectionSelected{} }
		case "h", "H":
			return m, func() tea.Msg { return GoHomeMsg{} }
		case "b", "esc":
			if m.fromSaved {
				return m, func() tea.Msg { return ConnectKnownSelected{} }
			}
			return m, func() tea.Msg { return GoHomeMsg{} }
		}
	}
	return m, nil
}

func (m *SessionErrorModel) View() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("Connection failed", "The port-forward session could not be started."))
	b.WriteString("\n\n")
	b.WriteString(ui.WarningStyle.Render(strings.ReplaceAll(m.failure.Message, "\n", "\n   ")))
	b.WriteString("\n\n")
	if m.failure.Detail != "" && m.failure.Detail != m.failure.Message {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.failure.Detail))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HintStyle.Render("Create a new connection to pick a different bastion, or go back and try another saved connection."))
	b.WriteString("\n\n")
	if m.fromSaved {
		b.WriteString(ui.HelpStyle.Render("n new connection · b saved connections · h home · q quit"))
	} else {
		b.WriteString(ui.HelpStyle.Render("n new connection · h home · q quit"))
	}
	return b.String()
}
