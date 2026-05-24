package steps

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/targetstore"
	"github.com/eichemberger/burrow/internal/ui"
)

type targetsRecoveryPhase int

const (
	targetsRecoveryWarn targetsRecoveryPhase = iota
	targetsRecoveryConfirm
	targetsRecoverySaving
)

type TargetsRecoveryModel struct {
	burrowDir string
	path      string
	loadErr   error
	phase     targetsRecoveryPhase
	err       string
	spinner   spinner.Model
}

func NewTargetsRecoveryModel(burrowDir string, loadErr error) *TargetsRecoveryModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = ui.PulseStyle

	return &TargetsRecoveryModel{
		burrowDir: burrowDir,
		path:      targetstore.TargetsPath(burrowDir),
		loadErr:   loadErr,
		phase:     targetsRecoveryWarn,
		spinner:   s,
	}
}

func (m *TargetsRecoveryModel) Init() tea.Cmd {
	return nil
}

func (m *TargetsRecoveryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case TargetsResetFailedMsg:
		m.phase = targetsRecoveryConfirm
		m.err = msg.Err.Error()
		return m, nil

	case tea.KeyMsg:
		switch m.phase {
		case targetsRecoveryWarn:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, func() tea.Msg { return QuitMsg{} }
			case "r", "R":
				m.phase = targetsRecoveryConfirm
				m.err = ""
				return m, nil
			}
		case targetsRecoveryConfirm:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, func() tea.Msg { return QuitMsg{} }
			case "b", "esc":
				m.phase = targetsRecoveryWarn
				m.err = ""
				return m, nil
			case "y", "Y":
				m.phase = targetsRecoverySaving
				m.err = ""
				return m, tea.Batch(m.spinner.Tick, m.resetTargets())
			}
		}
	}

	if m.phase == targetsRecoverySaving {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *TargetsRecoveryModel) resetTargets() tea.Cmd {
	return func() tea.Msg {
		store, err := targetstore.Reset(m.burrowDir)
		if err != nil {
			return TargetsResetFailedMsg{Err: err}
		}
		return TargetsResetCompleteMsg{Store: store}
	}
}

func (m *TargetsRecoveryModel) View() string {
	switch m.phase {
	case targetsRecoveryConfirm:
		return m.viewConfirm()
	case targetsRecoverySaving:
		return m.viewSaving()
	default:
		return m.viewWarn()
	}
}

func (m *TargetsRecoveryModel) viewWarn() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading(
		"Saved connections file is invalid",
		"burrow could not read your saved connections.",
	))
	b.WriteString("\n\n")
	b.WriteString(ui.FieldLabelStyle.Render("File"))
	b.WriteString("\n")
	b.WriteString(ui.HintStyle.Render(m.path))
	b.WriteString("\n\n")
	b.WriteString(ui.ErrorStyle.Render("✗ " + m.loadErr.Error()))
	b.WriteString("\n\n")
	b.WriteString(ui.HintStyle.Render(
		"Open this file in a text editor and fix the YAML syntax, then restart burrow.",
	))
	b.WriteString("\n\n")
	b.WriteString(ui.SubtitleStyle.Render("Or reset the file to start fresh (see next step)."))
	b.WriteString("\n\n")
	b.WriteString(ui.HelpStyle.Render("r reset file · q quit"))
	return b.String()
}

func (m *TargetsRecoveryModel) viewConfirm() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading(
		"Reset saved connections?",
		"This replaces targets.yaml with a new empty file.",
	))
	b.WriteString("\n\n")
	b.WriteString(ui.WarningStyle.Render(
		"⚠  WARNING: This permanently deletes ALL saved connections.\n" +
			"   You will lose every alias and connection you had configured.\n" +
			"   This cannot be undone.",
	))
	b.WriteString("\n\n")
	b.WriteString(ui.HintStyle.Render(fmt.Sprintf("A brand-new file will be written to:\n  %s", m.path)))
	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}
	b.WriteString(ui.HelpStyle.Render("y confirm reset · b back · q quit"))
	return b.String()
}

func (m *TargetsRecoveryModel) viewSaving() string {
	var b strings.Builder
	b.WriteString(ui.PageHeading("Resetting saved connections", m.path))
	b.WriteString("\n")
	b.WriteString(ui.LoadingLine(m.spinner.View(), "Creating new targets.yaml..."))
	return b.String()
}

type TargetsResetCompleteMsg struct {
	Store *targetstore.Store
}

type TargetsResetFailedMsg struct {
	Err error
}
