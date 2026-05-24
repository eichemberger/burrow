package steps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

type TargetEditModel struct {
	store    *targetstore.Store
	oldAlias string
	labels   []string
	inputs   []textinput.Model
	focused  int
	err      string
}

func NewTargetEditModel(store *targetstore.Store, alias string, target targetstore.Target) *TargetEditModel {
	useEnv := "false"
	profile := target.AWSProfile
	if target.UseEnv {
		useEnv = "true"
		profile = ""
	}

	values := []string{
		alias,
		useEnv,
		profile,
		target.Region,
		target.BastionID,
		target.Host,
		strconv.Itoa(target.RemotePort),
		strconv.Itoa(target.LocalPort),
		target.Description,
	}
	labels := []string{
		"Alias",
		"Use env (true/false)",
		"AWS profile",
		"Region",
		"Bastion instance ID",
		"Remote host",
		"Remote port",
		"Local port",
		"Description (optional)",
	}

	inputs := make([]textinput.Model, len(values))
	for i, value := range values {
		ti := textinput.New()
		ti.SetValue(value)
		ti.CharLimit = 255
		ti.Width = 50
		if i == 6 || i == 7 {
			ti.CharLimit = 5
			ti.Width = 10
		}
		if i == 1 {
			ti.Width = 10
			ti.CharLimit = 5
		}
		inputs[i] = ti
	}
	inputs[0].Focus()

	return &TargetEditModel{
		store:    store,
		oldAlias: alias,
		labels:   labels,
		inputs:   inputs,
	}
}

func (m *TargetEditModel) Init() tea.Cmd { return textinput.Blink }

func (m *TargetEditModel) inputValues() []string {
	values := make([]string, len(m.inputs))
	for i, input := range m.inputs {
		values[i] = input.Value()
	}
	return values
}

func (m *TargetEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			target, alias, err := m.buildTarget()
			if err != nil {
				m.err = err.Error()
				return m, nil
			}
			if m.oldAlias == "" {
				if err := m.store.Set(alias, target); err != nil {
					m.err = err.Error()
					return m, nil
				}
			} else {
				if alias != m.oldAlias {
					if err := m.store.Rename(m.oldAlias, alias); err != nil {
						m.err = err.Error()
						return m, nil
					}
				}
				if err := m.store.Set(alias, target); err != nil {
					m.err = err.Error()
					return m, nil
				}
			}
			return m, func() tea.Msg { return TargetSavedMsg{Alias: alias} }
		}
	}

	var cmd tea.Cmd
	if m.focused == 6 || m.focused == 7 {
		m.inputs[m.focused], cmd = updatePortInput(m.inputs[m.focused], msg)
	} else {
		m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	}
	return m, cmd
}

func (m *TargetEditModel) buildTarget() (targetstore.Target, string, error) {
	alias := strings.TrimSpace(m.inputs[0].Value())
	if err := targetstore.ValidateAlias(alias); err != nil {
		return targetstore.Target{}, "", err
	}

	useEnv := strings.EqualFold(strings.TrimSpace(m.inputs[1].Value()), "true")
	profile := strings.TrimSpace(m.inputs[2].Value())
	region := strings.TrimSpace(m.inputs[3].Value())
	bastionID := strings.TrimSpace(m.inputs[4].Value())
	host := strings.TrimSpace(m.inputs[5].Value())
	remotePort, err := strconv.Atoi(strings.TrimSpace(m.inputs[6].Value()))
	if err != nil {
		return targetstore.Target{}, "", fmt.Errorf("invalid remote port")
	}
	localPort, err := strconv.Atoi(strings.TrimSpace(m.inputs[7].Value()))
	if err != nil {
		return targetstore.Target{}, "", fmt.Errorf("invalid local port")
	}

	target := targetstore.Target{
		AWSProfile:  profile,
		UseEnv:      useEnv,
		Region:      region,
		BastionID:   bastionID,
		Host:        host,
		RemotePort:  remotePort,
		LocalPort:   localPort,
		Description: strings.TrimSpace(m.inputs[8].Value()),
	}
	if err := target.Validate(); err != nil {
		return targetstore.Target{}, "", err
	}
	return target, alias, nil
}

func (m *TargetEditModel) View() string {
	title := "Edit connection"
	subtitle := "Update the saved configuration"
	if m.oldAlias == "" {
		title = "Add connection"
		subtitle = "Create a new saved configuration"
	}

	var b strings.Builder
	b.WriteString(ui.PageHeading(title, subtitle))
	b.WriteString("\n")
	for i, label := range m.labels {
		b.WriteString(ui.FieldLabelStyle.Render(label))
		b.WriteString("\n")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("tab next field · enter save · " + ui.InputHelpKeys))
	return b.String()
}
