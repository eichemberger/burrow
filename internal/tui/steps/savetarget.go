package steps

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

type SaveTargetModel struct {
	inputs  []textinput.Model
	focused int
	store   *targetstore.Store
	err     string
}

func NewSaveTargetModel(store *targetstore.Store) *SaveTargetModel {
	alias := textinput.New()
	alias.Placeholder = "my-db-prod"
	alias.Focus()
	alias.CharLimit = 64
	alias.Width = 30

	description := textinput.New()
	description.Placeholder = "Production Postgres via bastion"
	description.CharLimit = 255
	description.Width = 50

	return &SaveTargetModel{
		inputs:  []textinput.Model{alias, description},
		focused: 0,
		store:   store,
	}
}

func (m *SaveTargetModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *SaveTargetModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleInputNavKeys(msg, m.inputs...); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab", "up", "down":
			if msg.String() == "shift+tab" || msg.String() == "up" {
				m.focused--
				if m.focused < 0 {
					m.focused = len(m.inputs) - 1
				}
			} else {
				m.focused = (m.focused + 1) % len(m.inputs)
			}
			for i := range m.inputs {
				if i == m.focused {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}
			return m, textinput.Blink

		case "enter":
			alias := strings.TrimSpace(m.inputs[0].Value())
			if alias == "" {
				return m, func() tea.Msg { return TargetSaveEntered{} }
			}
			if err := targetstore.ValidateAlias(alias); err != nil {
				m.err = err.Error()
				m.focused = 0
				m.inputs[0].Focus()
				m.inputs[1].Blur()
				return m, textinput.Blink
			}
			return m, func() tea.Msg {
				return TargetSaveEntered{
					Alias:       alias,
					Description: strings.TrimSpace(m.inputs[1].Value()),
				}
			}
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m *SaveTargetModel) View() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("Save this connection?", "Optional alias stored in ~/.burrow/targets.yaml"))
	b.WriteString("\n")
	b.WriteString(ui.FieldLabelStyle.Render("Alias"))
	b.WriteString("\n")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")
	b.WriteString(ui.FieldLabelStyle.Render("Description (optional)"))
	b.WriteString("\n")
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")
	b.WriteString(ui.HintStyle.Render("Leave alias empty and press Enter to skip and just connect."))
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("tab switch field · enter save & connect · " + ui.InputHelpKeys))
	return b.String()
}
