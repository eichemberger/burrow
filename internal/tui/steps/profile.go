package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/awsconfig"
	"github.com/eichemberger/burrow/internal/ui"
)

type ProfileModel struct {
	phase    int
	awsDir   string
	profiles []string
	list     list.Model
	width    int
	height   int
	err      error
}

func NewProfileModel(awsDir string, presetProfile string) (*ProfileModel, tea.Cmd) {
	m := &ProfileModel{
		awsDir: awsDir,
		phase:  0,
		list: newList([]stringItem{
			{value: "env", title: "Use environment variables", desc: "AWS_ACCESS_KEY_ID, AWS_PROFILE, etc."},
			{value: "profile", title: "Use AWS profile", desc: fmt.Sprintf("Load from %s", awsDir)},
		}, "AWS credentials", 6),
	}

	if presetProfile != "" {
		profiles := []stringItem{{value: presetProfile, title: presetProfile}}
		m.phase = 1
		m.list = newList(profiles, "Select AWS profile", 4)
	}

	return m, nil
}

func (m *ProfileModel) Init() tea.Cmd { return nil }

func (m *ProfileModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list = applyListSize(m.list, width, height)
}

func (m *ProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if cmd, handled := handleNavKeys(msg); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if m.phase == 0 {
				item := m.list.SelectedItem().(stringItem)
				if item.value == "env" {
					return m, func() tea.Msg { return AuthModeSelected{UseEnv: true} }
				}
				profiles, err := awsconfig.ListProfiles(m.awsDir)
				if err != nil {
					m.err = err
					return m, nil
				}
				m.profiles = profiles
				items := make([]stringItem, len(profiles))
				for i, p := range profiles {
					items[i] = stringItem{value: p, title: p}
				}
				m.list = newList(items, "Select AWS profile", min(12, len(items)+2))
				if m.width > 0 {
					m.list = applyListSize(m.list, m.width, m.height)
				}
				m.phase = 1
				return m, nil
			}

			profile := m.list.SelectedItem().(stringItem).value
			return m, func() tea.Msg { return ProfileSelected{Profile: profile} }
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *ProfileModel) View() string {
	var b strings.Builder
	if m.err != nil {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err.Error()))
		b.WriteString("\n\n")
	}
	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.HelpKeys))
	return b.String()
}
