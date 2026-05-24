package steps

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/awsconfig"
	"github.com/eichemberger/burrow/internal/ui"
)

type RegionModel struct {
	input textinput.Model
	err   string
}

func NewRegionModel(awsDir, profile string, presetRegion string) *RegionModel {
	ti := textinput.New()
	ti.Placeholder = "us-east-1"
	ti.Focus()
	ti.CharLimit = 32
	ti.Width = 40

	defaultRegion := presetRegion
	if defaultRegion == "" && profile != "" {
		defaultRegion = awsconfig.ProfileRegion(awsDir, profile)
	}
	if defaultRegion == "" {
		defaultRegion = "us-east-1"
	}
	ti.SetValue(defaultRegion)

	return &RegionModel{input: ti}
}

func (m *RegionModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *RegionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleInputNavKeys(msg, m.input); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			region := strings.TrimSpace(m.input.Value())
			if region == "" {
				m.err = "region is required"
				return m, nil
			}
			return m, func() tea.Msg { return RegionSelected{Region: region} }
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *RegionModel) View() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("AWS Region", "Used for API calls and the SSM session"))
	b.WriteString("\n")
	b.WriteString(ui.FieldLabelStyle.Render("Region"))
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
