package steps

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ui"
)

type ManualModel struct {
	inputs  []textinput.Model
	focused int
	err     string
}

func NewManualModel() *ManualModel {
	host := textinput.New()
	host.Placeholder = "my-db.cluster-abc123.us-east-1.rds.amazonaws.com"
	host.Focus()
	host.CharLimit = 255
	host.Width = 60

	port := textinput.New()
	port.Placeholder = "5432"
	port.CharLimit = 5
	port.Width = 10

	return &ManualModel{
		inputs:  []textinput.Model{host, port},
		focused: 0,
	}
}

func (m *ManualModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ManualModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			host := strings.TrimSpace(m.inputs[0].Value())
			portStr := strings.TrimSpace(m.inputs[1].Value())
			if host == "" {
				m.err = "host is required"
				return m, nil
			}
			port, err := strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				m.err = "invalid remote port"
				return m, nil
			}
			target := services.Target{
				Label: host,
				Host:  host,
				Port:  port,
			}
			return m, func() tea.Msg { return ManualTargetEntered{Target: target} }
		}
	}

	var cmd tea.Cmd
	if m.focused == 1 {
		m.inputs[m.focused], cmd = updatePortInput(m.inputs[m.focused], msg)
	} else {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	}
	return m, cmd
}

func (m *ManualModel) View() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("Manual connection", "Enter the remote host and port to forward to"))
	b.WriteString("\n")
	b.WriteString(ui.FieldLabelStyle.Render("Host"))
	b.WriteString("\n")
	b.WriteString(m.inputs[0].View())
	b.WriteString("\n\n")
	b.WriteString(ui.FieldLabelStyle.Render("Remote port"))
	b.WriteString("\n")
	b.WriteString(m.inputs[1].View())
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("tab switch field · " + ui.InputHelpKeys))
	return b.String()
}
