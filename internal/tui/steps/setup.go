package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/configstore"
	"github.com/eichemberger/burrow/internal/ui"
)

type setupPhase int

const (
	setupIntro setupPhase = iota
	setupTagKey
	setupTagValue
	setupAddMore
	setupSaving
)

type SetupModel struct {
	burrowDir  string
	invalid    bool
	phase      setupPhase
	tagFilters []configstore.TagFilter
	pendingKey string
	keyInput   textinput.Model
	valueInput textinput.Model
	spinner    spinner.Model
	err        string
}

func NewSetupModel(burrowDir string, configErr error) *SetupModel {
	keyInput := textinput.New()
	keyInput.Placeholder = "Role"
	keyInput.CharLimit = 128
	keyInput.Width = 40

	valueInput := textinput.New()
	valueInput.Placeholder = "bastion"
	valueInput.CharLimit = 256
	valueInput.Width = 40

	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = ui.PulseStyle

	return &SetupModel{
		burrowDir:  burrowDir,
		invalid:    configstore.IsInvalid(configErr),
		phase:      setupIntro,
		keyInput:   keyInput,
		valueInput: valueInput,
		spinner:    s,
	}
}

func (m *SetupModel) Init() tea.Cmd {
	return nil
}

func (m *SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case SetupSavedMsg:
		m.phase = setupIntro
		m.err = ""
		return m, func() tea.Msg { return SetupCompleteMsg{Config: msg.Config} }

	case SetupSaveFailedMsg:
		m.phase = setupAddMore
		m.err = msg.Err.Error()
		return m, nil
	}

	switch m.phase {
	case setupIntro:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return m.updateIntro(keyMsg)
		}
	case setupTagKey:
		return m.updateTagKey(msg)
	case setupTagValue:
		return m.updateTagValue(msg)
	case setupAddMore:
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return m.updateAddMore(keyMsg)
		}
	case setupSaving:
		return m.updateSaving(msg)
	}

	return m, nil
}

func (m *SetupModel) updateIntro(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, func() tea.Msg { return QuitMsg{} }
	case "enter":
		m.phase = setupTagKey
		m.keyInput.Focus()
		return m, textinput.Blink
	}
	return m, nil
}

func (m *SetupModel) updateTagKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if cmd, handled := handleInputNavKeys(keyMsg, m.keyInput, m.valueInput); handled {
			return m, cmd
		}
		switch keyMsg.String() {
		case "enter":
			key := strings.TrimSpace(m.keyInput.Value())
			if key == "" {
				m.err = "tag key is required"
				return m, nil
			}
			m.pendingKey = key
			m.err = ""
			m.phase = setupTagValue
			m.valueInput.SetValue("")
			m.valueInput.Focus()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	return m, cmd
}

func (m *SetupModel) updateTagValue(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if cmd, handled := handleInputNavKeys(keyMsg, m.keyInput, m.valueInput); handled {
			return m, cmd
		}
		switch keyMsg.String() {
		case "enter":
			value := strings.TrimSpace(m.valueInput.Value())
			if value == "" {
				m.err = "tag value is required"
				return m, nil
			}
			m.tagFilters = append(m.tagFilters, configstore.TagFilter{
				Key:   m.pendingKey,
				Value: value,
			})
			m.pendingKey = ""
			m.err = ""
			m.phase = setupAddMore
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.valueInput, cmd = m.valueInput.Update(msg)
	return m, cmd
}

func (m *SetupModel) updateAddMore(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, func() tea.Msg { return QuitMsg{} }
	case "y", "Y":
		m.phase = setupTagKey
		m.keyInput.SetValue("")
		m.keyInput.Focus()
		return m, textinput.Blink
	case "n", "N", "enter":
		m.phase = setupSaving
		m.err = ""
		return m, tea.Batch(m.spinner.Tick, m.saveConfig())
	}
	return m, nil
}

func (m *SetupModel) updateSaving(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m *SetupModel) saveConfig() tea.Cmd {
	return func() tea.Msg {
		cfg := configstore.NewEC2Config(m.tagFilters)
		if err := configstore.Save(m.burrowDir, cfg); err != nil {
			return SetupSaveFailedMsg{Err: err}
		}
		loaded, err := configstore.Load(m.burrowDir)
		if err != nil {
			return SetupSaveFailedMsg{Err: err}
		}
		return SetupSavedMsg{Config: loaded}
	}
}

func (m *SetupModel) View() string {
	switch m.phase {
	case setupIntro:
		return m.viewIntro()
	case setupTagKey:
		return m.viewTagKey()
	case setupTagValue:
		return m.viewTagValue()
	case setupAddMore:
		return m.viewAddMore()
	case setupSaving:
		return m.viewSaving()
	default:
		return ""
	}
}

func (m *SetupModel) viewIntro() string {
	var b strings.Builder
	title := "Welcome to burrow"
	subtitle := "One-time setup: choose EC2 tags that identify your SSM bastion instances."
	if m.invalid {
		subtitle = "Your config file is missing or invalid. Let's set it up again."
	}
	b.WriteString(ui.PageHeading(title, subtitle))
	b.WriteString("\n\n")
	b.WriteString(ui.HintStyle.Render("Instances must match ALL configured tag filters (key=value)."))
	b.WriteString("\n\n")
	b.WriteString(ui.HelpStyle.Render("enter continue · q quit"))
	return b.String()
}

func (m *SetupModel) viewTagKey() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("EC2 tag filter", "Enter the tag key for a bastion instance."))
	b.WriteString("\n")
	b.WriteString(ui.FieldLabelStyle.Render("Tag key"))
	b.WriteString("\n")
	b.WriteString(m.keyInput.View())
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("enter next · " + ui.InputHelpKeys))
	return b.String()
}

func (m *SetupModel) viewTagValue() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("EC2 tag filter", fmt.Sprintf("Enter the value for tag %q.", m.pendingKey)))
	b.WriteString("\n")
	b.WriteString(ui.FieldLabelStyle.Render("Tag value"))
	b.WriteString("\n")
	b.WriteString(m.valueInput.View())
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("enter add filter · " + ui.InputHelpKeys))
	return b.String()
}

func (m *SetupModel) viewAddMore() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("EC2 tag filters", "Configured filters so far:"))
	b.WriteString("\n\n")
	for _, f := range m.tagFilters {
		b.WriteString(ui.HintStyle.Render(fmt.Sprintf("  • %s = %s", f.Key, f.Value)))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(ui.SubtitleStyle.Render("Add another tag filter?"))
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("y add another · n or enter save · q quit"))
	return b.String()
}

func (m *SetupModel) viewSaving() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("Saving configuration", fmt.Sprintf("Writing %s", configstore.ConfigPath(m.burrowDir))))
	b.WriteString("\n")
	b.WriteString(ui.LoadingLine(m.spinner.View(), "Saving config.yaml..."))
	return b.String()
}

type SetupSavedMsg struct {
	Config *configstore.Config
}

type SetupSaveFailedMsg struct {
	Err error
}
