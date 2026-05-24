package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Adaptive palette: looks right on both light and dark terminals.
var (
	Accent    = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"}
	Accent2   = lipgloss.AdaptiveColor{Light: "#DB2777", Dark: "#F472B6"}
	Highlight = lipgloss.AdaptiveColor{Light: "#0EA5E9", Dark: "#22D3EE"}
	Success   = lipgloss.AdaptiveColor{Light: "#16A34A", Dark: "#4ADE80"}
	Warning   = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"}
	Danger    = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}

	TextColor   = lipgloss.AdaptiveColor{Light: "#0F172A", Dark: "#F1F5F9"}
	MutedColor  = lipgloss.AdaptiveColor{Light: "#475569", Dark: "#94A3B8"}
	SubtleColor = lipgloss.AdaptiveColor{Light: "#94A3B8", Dark: "#64748B"}
	FaintColor  = lipgloss.AdaptiveColor{Light: "#CBD5E1", Dark: "#334155"}
)

// Common styles. Kept minimal: no heavy borders that eat row height.
var (
	BrandStyle = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	PulseStyle = lipgloss.NewStyle().Bold(true).Foreground(Accent2)

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextColor).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().Foreground(MutedColor)

	HelpStyle = lipgloss.NewStyle().Foreground(SubtleColor)

	ErrorStyle   = lipgloss.NewStyle().Foreground(Danger).Bold(true)
	WarningStyle = lipgloss.NewStyle().Foreground(Warning).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(Success).Bold(true)

	DimStyle      = lipgloss.NewStyle().Foreground(SubtleColor)
	SelectedStyle = lipgloss.NewStyle().Foreground(Accent).Bold(true)

	FieldLabelStyle = lipgloss.NewStyle().
			Foreground(Highlight).
			Bold(true).
			MarginBottom(0)

	HintStyle = lipgloss.NewStyle().Foreground(MutedColor).Italic(true)
)

// Help text shown at the bottom of every screen.
var (
	HelpKeys       = formatHelp([]string{"↑/↓", "navigate"}, []string{"enter", "select"}, []string{"b", "back"}, []string{"q", "quit"})
	SearchHelpKeys = formatHelp([]string{"↑/↓", "navigate"}, []string{"/", "search"}, []string{"enter", "select"}, []string{"esc", "cancel"}, []string{"b", "back"}, []string{"q", "quit"})
	InputHelpKeys  = formatHelp([]string{"enter", "confirm"}, []string{"esc", "back"}, []string{"ctrl+c", "quit"})
)

func formatHelp(pairs ...[]string) string {
	keyStyle := lipgloss.NewStyle().Foreground(Highlight).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(SubtleColor)
	parts := make([]string, 0, len(pairs))
	for _, p := range pairs {
		parts = append(parts, keyStyle.Render(p[0])+" "+descStyle.Render(p[1]))
	}
	return strings.Join(parts, lipgloss.NewStyle().Foreground(FaintColor).Render(" · "))
}

// Layout constants for the page chrome.
const (
	maxPageWidth = 96
	sidePadding  = 4
	topPadding   = 1
)

// PageDimensions returns the inner content (width, height) available to a step's View.
// The page reserves space for the header, optional breadcrumb, and outer padding.
func PageDimensions(termWidth, termHeight int, hasBreadcrumb bool) (int, int) {
	w := termWidth - sidePadding*2
	if w > maxPageWidth {
		w = maxPageWidth
	}
	if w < 30 {
		w = 30
	}
	chrome := topPadding + 2 /* header + rule */ + 1 /* spacing */
	if hasBreadcrumb {
		chrome += 2 // breadcrumb + spacing
	}
	chrome += 1 // bottom buffer
	h := termHeight - chrome
	if h < 8 {
		h = 8
	}
	return w, h
}

// Page wraps a child view with the brand header and an optional wizard breadcrumb,
// then centers it horizontally in the terminal viewport.
func Page(termWidth, termHeight, pulse int, breadcrumb, body string) string {
	if termWidth <= 0 || termHeight <= 0 {
		return body
	}

	w := termWidth - sidePadding*2
	if w > maxPageWidth {
		w = maxPageWidth
	}
	if w < 30 {
		w = 30
	}

	header := renderHeader(w, pulse)

	parts := []string{header, ""}
	if breadcrumb != "" {
		parts = append(parts, breadcrumb, "")
	}
	parts = append(parts, body)

	block := lipgloss.JoinVertical(lipgloss.Left, parts...)
	block = lipgloss.NewStyle().Width(w).Render(block)
	block = lipgloss.NewStyle().PaddingTop(topPadding).Render(block)

	return lipgloss.Place(termWidth, termHeight, lipgloss.Center, lipgloss.Top, block)
}

func renderHeader(width, pulse int) string {
	pulseGlyph := PulseStyle.Render(currentPulseGlyph(pulse))
	tagline := SubtitleStyle.Render("AWS SSM port forwarding made easy")
	left := fmt.Sprintf("%s  %s  %s", BrandStyle.Render("🐰 BURROW"), pulseGlyph, tagline)
	rule := lipgloss.NewStyle().Foreground(FaintColor).Render(strings.Repeat("─", width))
	return lipgloss.JoinVertical(lipgloss.Left, left, rule)
}

// LoadingLine renders an inline spinner+label for in-flight operations.
func LoadingLine(spinnerView, text string) string {
	return PulseStyle.Render(spinnerView) + "  " + SubtitleStyle.Render(text)
}

// SectionHeading renders a screen-level heading with a soft accent rule below.
func SectionHeading(text string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(Accent).Render(text),
		"",
	)
}

// PageHeading renders a screen-level heading with an optional subtitle. Use
// this on screens without a list (text inputs, confirmations) so every screen
// has consistent typography.
func PageHeading(title, subtitle string) string {
	heading := lipgloss.NewStyle().
		Bold(true).
		Foreground(TextColor).
		Render(title)

	if subtitle == "" {
		return heading + "\n"
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		heading,
		SubtitleStyle.Render(subtitle),
	) + "\n"
}

func currentPulseGlyph(pulse int) string {
	glyphs := []string{"○", "◔", "◑", "◕", "●", "◕", "◑", "◔"}
	return glyphs[pulse%len(glyphs)]
}

// ─── Wizard breadcrumb ──────────────────────────────────────────────

type WizardStep int

const (
	WizardNone WizardStep = iota
	WizardAuth
	WizardRegion
	WizardService
	WizardResource
	WizardEndpoint
	WizardBastion
	WizardLocalPort
	WizardSave
	WizardConnect
)

var wizardOrder = []WizardStep{
	WizardAuth,
	WizardRegion,
	WizardService,
	WizardResource,
	WizardBastion,
	WizardLocalPort,
	WizardSave,
	WizardConnect,
}

var wizardLabel = map[WizardStep]string{
	WizardAuth:      "auth",
	WizardRegion:    "region",
	WizardService:   "service",
	WizardResource:  "resource",
	WizardEndpoint:  "endpoint",
	WizardBastion:   "bastion",
	WizardLocalPort: "port",
	WizardSave:      "save",
	WizardConnect:   "connect",
}

// WizardBreadcrumb renders a step indicator with the current step highlighted.
// Returns an empty string for non-wizard steps.
func WizardBreadcrumb(current WizardStep) string {
	if current == WizardNone {
		return ""
	}

	currentIdx := -1
	for i, step := range wizardOrder {
		if step == current {
			currentIdx = i
			break
		}
	}

	doneStyle := lipgloss.NewStyle().Foreground(Success)
	currentStyle := lipgloss.NewStyle().Foreground(Accent).Bold(true)
	pendingStyle := lipgloss.NewStyle().Foreground(SubtleColor)
	sepStyle := lipgloss.NewStyle().Foreground(FaintColor)

	parts := make([]string, 0, len(wizardOrder))
	for i, step := range wizardOrder {
		label := wizardLabel[step]
		switch {
		case i == currentIdx:
			parts = append(parts, currentStyle.Render("● "+label))
		case currentIdx > 0 && i < currentIdx:
			parts = append(parts, doneStyle.Render("✓ "+label))
		default:
			parts = append(parts, pendingStyle.Render("○ "+label))
		}
	}

	return strings.Join(parts, sepStyle.Render(" › "))
}
