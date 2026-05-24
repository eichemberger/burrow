package steps

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/apictx"
	"github.com/eichemberger/burrow/internal/awsconfig"
	"github.com/eichemberger/burrow/internal/netutil"
	"github.com/eichemberger/burrow/internal/ssmexec"
	"github.com/eichemberger/burrow/internal/ui"
)

type runPhase int

const (
	runPreflight runPhase = iota
	runExec
)

type RunModel struct {
	awsDir    string
	opts      ssmexec.Options
	fromSaved bool
	summary   string
	phase     runPhase
	spinner   spinner.Model
}

func NewRunModel(awsDir string, opts ssmexec.Options, fromSaved bool) *RunModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = ui.PulseStyle

	return &RunModel{
		awsDir:    awsDir,
		opts:      opts,
		fromSaved: fromSaved,
		summary:   ssmexec.Summary(opts),
		phase:     runPreflight,
		spinner:   s,
	}
}

func (m *RunModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.preflight())
}

func (m *RunModel) preflight() tea.Cmd {
	return func() tea.Msg {
		authCtx, authCancel := apictx.AuthBackground()
		defer authCancel()

		cfg, err := awsconfig.Load(authCtx, awsconfig.Options{
			AWSDir:  m.awsDir,
			Profile: m.opts.Profile,
			Region:  m.opts.Region,
			UseEnv:  m.opts.UseEnv,
		})
		if err != nil {
			return RunPreflightMsg{Err: apictx.WrapDeadline(err, "load AWS configuration")}
		}

		ctx, cancel := apictx.Background()
		defer cancel()

		err = ssmexec.VerifyInstanceOnline(ctx, cfg, m.opts.TargetInstanceID)
		if err != nil {
			return RunPreflightMsg{Err: err}
		}
		return RunPreflightMsg{Err: netutil.LocalPortAvailable(m.opts.LocalPort)}
	}
}

func (m *RunModel) startExec() tea.Cmd {
	cmd, err := ssmexec.BuildCommand(m.opts)
	if err != nil {
		return func() tea.Msg {
			return RunFailedMsg{
				Failure:   ssmexec.ClassifyRunError(m.opts.TargetInstanceID, "", err),
				FromSaved: m.fromSaved,
			}
		}
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return RunFailedMsg{
				Failure:   ssmexec.ClassifyRunError(m.opts.TargetInstanceID, stderr.String(), err),
				FromSaved: m.fromSaved,
			}
		}
		return RunFinished{}
	})
}

func (m *RunModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RunPreflightMsg:
		if msg.Err != nil {
			return m, func() tea.Msg {
				return RunFailedMsg{
					Failure:   toRunFailure(msg.Err, m.opts.TargetInstanceID),
					FromSaved: m.fromSaved,
				}
			}
		}
		m.phase = runExec
		return m, m.startExec()

	case RunFinished:
		return m, tea.Quit
	}

	if m.phase == runPreflight {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	if cmd, handled := handleNavKeys(msg); handled {
		return m, cmd
	}

	return m, nil
}

func (m *RunModel) View() string {
	var b strings.Builder
	if m.phase == runPreflight {
		b.WriteString(ui.PageHeading("Checking bastion", "Verifying SSM connectivity before starting the session"))
		b.WriteString("\n")
		b.WriteString(ui.SubtitleStyle.Render(m.summary))
		b.WriteString("\n\n")
		b.WriteString(ui.LoadingLine(m.spinner.View(), "Checking bastion and local port availability..."))
		return b.String()
	}

	b.WriteString(ui.PageHeading("Starting port forward", "Handing off to the AWS CLI"))
	b.WriteString("\n")
	b.WriteString(ui.SubtitleStyle.Render(m.summary))
	b.WriteString("\n\n")
	b.WriteString(ui.HintStyle.Render("Press Ctrl+C in the session to stop."))
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.HelpKeys))
	return b.String()
}

func toRunFailure(err error, instanceID string) ssmexec.RunFailure {
	var failure ssmexec.RunFailure
	if errors.As(err, &failure) {
		return failure
	}
	return ssmexec.ClassifyRunError(instanceID, err.Error(), err)
}

type RunPreflightMsg struct {
	Err error
}
