package steps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/netutil"
	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ui"
)

type LocalPortModel struct {
	input  textinput.Model
	target services.Target
	err    string
}

func NewLocalPortModel(target services.Target) *LocalPortModel {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("%d", target.Port)
	ti.Focus()
	ti.CharLimit = 5
	ti.Width = 10
	if target.Port > 0 {
		ti.SetValue(strconv.Itoa(target.Port))
	}

	return &LocalPortModel{input: ti, target: target}
}

func (m *LocalPortModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *LocalPortModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleInputNavKeys(msg, m.input); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			portStr := strings.TrimSpace(m.input.Value())
			port, err := strconv.Atoi(portStr)
			if err != nil || port < 1 || port > 65535 {
				m.err = "invalid local port"
				return m, nil
			}
			if err := netutil.LocalPortAvailable(port); err != nil {
				m.err = err.Error()
				return m, nil
			}
			return m, func() tea.Msg { return LocalPortEntered{Port: port} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = updatePortInput(m.input, msg)
	return m, cmd
}

func (m *LocalPortModel) View() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading(
		"Local port",
		fmt.Sprintf("localhost:<port>  →  %s:%d", m.target.Host, m.target.Port),
	))
	b.WriteString("\n")
	b.WriteString(ui.FieldLabelStyle.Render("Port"))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render(ui.InputHelpKeys))
	return b.String()
}
