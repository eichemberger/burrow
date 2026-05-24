package steps

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/eichemberger/burrow/internal/apictx"
	"github.com/eichemberger/burrow/internal/bastion"
	"github.com/eichemberger/burrow/internal/configstore"
	"github.com/eichemberger/burrow/internal/debuglog"
	"github.com/eichemberger/burrow/internal/netutil"
	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ui"
)

type BastionModel struct {
	cfg            aws.Config
	target         services.Target
	ec2Filter      *configstore.EC2Selector
	list           list.Model
	spinner        spinner.Model
	loading        bool
	listReady      bool
	bastions       []bastion.Instance
	globalWarnings []string

	pending        *bastion.Instance
	confirmWarning string

	width  int
	height int
	err    error
}

func NewBastionModel(cfg aws.Config, target services.Target, ec2Filter *configstore.EC2Selector) *BastionModel {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	s.Style = ui.PulseStyle

	return &BastionModel{
		cfg:       cfg,
		target:    target,
		ec2Filter: ec2Filter,
		spinner:   s,
		loading:   true,
	}
}

func (m *BastionModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadBastions())
}

func (m *BastionModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.listReady {
		m.list = applyListSize(m.list, width, height)
	}
}

func (m *BastionModel) loadBastions() tea.Cmd {
	return func() tea.Msg {
		if m.ec2Filter != nil {
			debuglog.Printf("loading bastions target=%s:%d ec2_tag_filters=%d", m.target.Host, m.target.Port, len(m.ec2Filter.TagFilters))
		} else {
			debuglog.Printf("loading bastions target=%s:%d ec2_tag_filters=none", m.target.Host, m.target.Port)
		}
		ctx, cancel := apictx.Background()
		defer cancel()
		result, err := bastion.ListReachable(ctx, m.cfg, m.target, m.ec2Filter)
		return BastionsLoadedMsg{Bastions: result.Instances, Warnings: result.Warnings, Err: err}
	}
}

func (m *BastionModel) confirming() bool {
	return m.pending != nil
}

func (m *BastionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.confirming() {
		return m.updateConfirm(msg)
	}

	if cmd, handled := handleNavKeysWithList(msg, m.list, m.listReady); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case BastionsLoadedMsg:
		m.loading = false
		m.err = msg.Err
		m.bastions = msg.Bastions
		m.globalWarnings = msg.Warnings
		debuglog.Printf("bastions loaded count=%d warnings=%d err=%v", len(msg.Bastions), len(msg.Warnings), msg.Err)
		if msg.Err != nil {
			return m, nil
		}
		items := make([]stringItem, len(msg.Bastions))
		for i, b := range msg.Bastions {
			desc := fmt.Sprintf("%s | %s | %s", b.ID, b.PrivateIP, b.State)
			title := b.Name
			if title == "" {
				title = b.ID
			}
			items[i] = stringItem{
				value:  fmt.Sprintf("%d", i),
				title:  title,
				desc:   desc,
				filter: strings.Join([]string{b.Name, b.ID, b.PrivateIP, title}, " "),
			}
		}
		m.list = newSearchableList(items, "Select SSM bastion instance", min(16, len(items)+2))
		m.listReady = true
		if m.width > 0 {
			m.list = applyListSize(m.list, m.width, m.height)
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "enter" && !m.loading && m.err == nil && m.list.FilterState() != list.Filtering {
			idx, ok := selectedItemIndex(m.list)
			if !ok || idx >= len(m.bastions) {
				return m, nil
			}
			selected := m.bastions[idx]
			if warning := selectionWarning(selected, m.globalWarnings); warning != "" {
				copy := selected
				m.pending = &copy
				m.confirmWarning = warning
				return m, nil
			}
			return m, func() tea.Msg { return BastionSelected{Bastion: selected} }
		}
	}

	var cmds []tea.Cmd
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.err == nil {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *BastionModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, func() tea.Msg { return QuitMsg{} }
		case "enter":
			selected := *m.pending
			m.pending = nil
			m.confirmWarning = ""
			return m, func() tea.Msg { return BastionSelected{Bastion: selected} }
		case "b", "esc":
			m.pending = nil
			m.confirmWarning = ""
			return m, nil
		}
	}
	return m, nil
}

func selectionWarning(inst bastion.Instance, global []string) string {
	var parts []string
	for _, w := range global {
		parts = append(parts, w)
	}
	if inst.AccessVia == bastion.AccessViaCIDR {
		name := inst.Name
		if name == "" {
			name = inst.ID
		}
		parts = append(parts, fmt.Sprintf(
			"%s (%s): access granted via CIDR rule (%s), not security group reference",
			name, inst.ID, inst.AccessNote,
		))
	}
	if inst.AccessNote == "security group validation skipped" {
		parts = append(parts, "Target security groups are unknown; connectivity was not fully validated.")
	}
	return strings.Join(parts, "\n")
}

func (m *BastionModel) View() string {
	if m.confirming() {
		return m.viewConfirm()
	}

	var b strings.Builder
	subtitle := bastionSubtitle(m.target.Host, m.target.Port, m.ec2Filter)

	if m.loading {
		b.WriteString(ui.PageHeading("Pick an SSM bastion", subtitle))
		b.WriteString("\n")
		b.WriteString(ui.LoadingLine(m.spinner.View(), "Evaluating reachable SSM instances..."))
	} else if m.err != nil {
		b.WriteString(ui.PageHeading("Pick an SSM bastion", subtitle))
		b.WriteString("\n")
		b.WriteString(ui.ErrorStyle.Render("✗ " + m.err.Error()))
	} else {
		b.WriteString(ui.SubtitleStyle.Render(subtitle))
		b.WriteString("\n\n")
		b.WriteString(m.list.View())
	}
	b.WriteString("\n")
	b.WriteString(ui.HelpStyle.Render(ui.SearchHelpKeys))
	return b.String()
}

func bastionSubtitle(host string, port int, ec2Filter *configstore.EC2Selector) string {
	if ec2Filter != nil && len(ec2Filter.TagFilters) > 0 {
		return fmt.Sprintf("Forwarding to %s:%d · instances matching configured EC2 tags", host, port)
	}

	label := "RFC 1918 private"
	if ec2Filter != nil {
		if nets, err := ec2Filter.PrivateNetworks(); err == nil {
			label = nets.Label()
		}
	} else if nets, err := netutil.DefaultPrivateNetworks(); err == nil {
		label = nets.Label()
	}
	return fmt.Sprintf("Forwarding to %s:%d · %s instances with SG access", host, port, label)
}

func (m *BastionModel) viewConfirm() string {
	name := m.pending.Name
	if name == "" {
		name = m.pending.ID
	}

	var b strings.Builder
	b.WriteString(ui.PageHeading("Confirm bastion", fmt.Sprintf("%s (%s)", name, m.pending.PrivateIP)))
	b.WriteString("\n")
	b.WriteString(ui.WarningStyle.Render("⚠  " + strings.ReplaceAll(m.confirmWarning, "\n", "\n   ")))
	b.WriteString("\n\n")
	b.WriteString(ui.HelpStyle.Render("enter continue · b back · q quit"))
	return b.String()
}
