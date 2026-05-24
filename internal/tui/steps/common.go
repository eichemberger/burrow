package steps

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/eichemberger/burrow/internal/services"
	"github.com/eichemberger/burrow/internal/ui"
)

const (
	defaultListWidth = 80
	// Rows the list reserves outside its viewport (title bar + paginator + breathing room).
	listChromeRows = 4
)

type stringItem struct {
	value  string
	title  string
	desc   string
	filter string
}

func (i stringItem) FilterValue() string {
	if i.filter != "" {
		return i.filter
	}
	return i.title + " " + i.desc
}
func (i stringItem) Title() string       { return i.title }
func (i stringItem) Description() string { return i.desc }

type Sizable interface {
	SetSize(width, height int)
}

func newList(items []stringItem, title string, height int) list.Model {
	return buildList(items, title, height, false)
}

func newSearchableList(items []stringItem, title string, height int) list.Model {
	return buildList(items, title, height, true)
}

func listItemsFrom[T list.Item](items []T) []list.Item {
	out := make([]list.Item, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

func newTargetList(items []list.Item, title string, minHeight, paginationAt int) list.Model {
	delegate := NewListDelegate()

	height := listMinHeight(len(items), delegate)
	if height < minHeight {
		height = minHeight
	}

	l := list.New(items, delegate, defaultListWidth, height)
	l.Title = title
	StyleList(&l)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetShowFilter(true)
	l.SetShowPagination(len(items) > paginationAt)
	return l
}

// newPlainList builds an unsearchable list with no inline title bar — meant for
// short menus where the surrounding page heading already supplies context.
func newPlainList(items []stringItem) list.Model {
	delegate := NewListDelegate()

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	height := listMinHeight(len(items), delegate)
	l := list.New(listItems, delegate, defaultListWidth, height)
	StyleList(&l)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(len(items) > 6)
	l.SetFilteringEnabled(false)
	return l
}

func buildList(items []stringItem, title string, height int, searchable bool) list.Model {
	delegate := NewListDelegate()

	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	if height < listMinHeight(len(items), delegate) {
		height = listMinHeight(len(items), delegate)
	}

	l := list.New(listItems, delegate, defaultListWidth, height)
	l.Title = title
	StyleList(&l)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(len(items) > 6)
	if searchable {
		l.SetFilteringEnabled(true)
		l.SetShowFilter(true)
	} else {
		l.SetFilteringEnabled(false)
	}
	return l
}

// NewListDelegate returns a 2-line (title + description) delegate styled with our palette.
// The selected row gets an accent rail on the left so focus is unmistakable.
func NewListDelegate() list.DefaultDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	delegate.SetSpacing(1)

	delegate.Styles.NormalTitle = lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Padding(0, 0, 0, 2)
	delegate.Styles.NormalDesc = lipgloss.NewStyle().
		Foreground(ui.MutedColor).
		Padding(0, 0, 0, 2)

	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(ui.Accent).
		Foreground(ui.Accent).
		Bold(true).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Border(lipgloss.ThickBorder(), false, false, false, true).
		BorderForeground(ui.Accent).
		Foreground(ui.MutedColor).
		Padding(0, 0, 0, 1)

	delegate.Styles.DimmedTitle = lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Padding(0, 0, 0, 2)
	delegate.Styles.DimmedDesc = lipgloss.NewStyle().
		Foreground(ui.FaintColor).
		Padding(0, 0, 0, 2)

	delegate.Styles.FilterMatch = lipgloss.NewStyle().
		Foreground(ui.Accent2).
		Bold(true).
		Underline(true)

	return delegate
}

// StyleList applies our title/pagination/filter styling to a list model.
// Importantly the title has NO bottom border so it doesn't steal viewport rows.
func StyleList(l *list.Model) {
	l.Styles.TitleBar = lipgloss.NewStyle().Padding(0, 0, 1, 0)
	l.Styles.Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(ui.Accent).
		Padding(0, 1)
	l.Styles.PaginationStyle = lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		PaddingLeft(2)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(ui.Accent2).Bold(true)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(ui.Accent)
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(ui.SubtleColor).Italic(true)
}

// listMinHeight returns the smallest list height that can render every item without truncation.
// height = titleBar(2) + items*delegate.Height + (items-1)*delegate.Spacing + paginator(1).
func listMinHeight(itemCount int, d list.DefaultDelegate) int {
	if itemCount <= 0 {
		itemCount = 1
	}
	rows := 2 // title bar
	rows += itemCount * d.Height()
	if itemCount > 1 {
		rows += (itemCount - 1) * d.Spacing()
	}
	rows += 1 // paginator/breathing room
	return rows
}

// applyListSize sets the list viewport based on the available page area.
func applyListSize(l list.Model, width, height int) list.Model {
	if width <= 0 {
		width = defaultListWidth
	}
	listHeight := height - listChromeRows
	if listHeight < 8 {
		listHeight = 8
	}
	l.SetSize(width, listHeight)
	return l
}

func resourceFilterText(r services.Resource) string {
	parts := []string{r.Label}
	for _, ep := range r.Endpoints {
		parts = append(parts, ep.Label, ep.Target.Host)
	}
	return strings.Join(parts, " ")
}

func endpointFilterText(ep services.Endpoint) string {
	return strings.Join([]string{ep.Label, ep.Target.Host, fmt.Sprintf("%d", ep.Target.Port)}, " ")
}

func selectedItemIndex(l list.Model) (int, bool) {
	item, ok := l.SelectedItem().(stringItem)
	if !ok {
		return 0, false
	}
	idx, err := strconv.Atoi(item.value)
	if err != nil {
		return 0, false
	}
	return idx, true
}

func handleNavKeys(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return func() tea.Msg { return QuitMsg{} }, true
		case "b", "esc":
			return func() tea.Msg { return BackMsg{} }, true
		}
	}
	return nil, false
}

// handleInputNavKeys is the single source of truth for nav shortcuts on any
// screen where a text input is focused. Only non-printable keys are reserved;
// every printable character is passed to the input. ESC navigates back only
// when all provided inputs are empty, so accidental ESC does not discard edits.
func handleInputNavKeys(msg tea.Msg, inputs ...textinput.Model) (tea.Cmd, bool) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.Type {
		case tea.KeyCtrlC:
			return func() tea.Msg { return QuitMsg{} }, true
		case tea.KeyEsc:
			if len(inputs) > 0 && !inputsEmpty(inputs) {
				return nil, false
			}
			return func() tea.Msg { return BackMsg{} }, true
		}
	}
	return nil, false
}

func inputsEmpty(inputs []textinput.Model) bool {
	for _, input := range inputs {
		if strings.TrimSpace(input.Value()) != "" {
			return false
		}
	}
	return true
}

func updatePortInput(input textinput.Model, msg tea.Msg) (textinput.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyRunes:
			filtered := filterDigitRunes(msg.Runes)
			if len(filtered) == 0 {
				return input, nil
			}
			if len(filtered) != len(msg.Runes) {
				msg.Runes = filtered
			}
		case tea.KeyBackspace, tea.KeyDelete, tea.KeyLeft, tea.KeyRight, tea.KeyHome, tea.KeyEnd, tea.KeyCtrlU, tea.KeyCtrlW, tea.KeyCtrlA, tea.KeyCtrlE:
			// allow editing/navigation
		default:
			return input, nil
		}
	}
	return input.Update(msg)
}

func filterDigitRunes(runes []rune) []rune {
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return out
}

func handleNavKeysWithList(msg tea.Msg, l list.Model, listReady bool) (tea.Cmd, bool) {
	if listReady && l.FilterState() == list.Filtering {
		// Same printable-safe contract as text inputs; esc cancels the filter
		// (handled by the list) rather than navigating back.
		if k, ok := msg.(tea.KeyMsg); ok && k.Type == tea.KeyCtrlC {
			return func() tea.Msg { return QuitMsg{} }, true
		}
		return nil, false
	}
	return handleNavKeys(msg)
}

func HandleQuitOnly(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return func() tea.Msg { return QuitMsg{} }, true
		}
	}
	return nil, false
}
