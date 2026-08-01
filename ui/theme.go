package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/zehnyth/cygnus/types"
)

// Palette. Every shade is an AdaptiveColor so the UI stays legible on
// both light and dark terminals — the Light value is a darker/richer
// variant that reads on cream/white backgrounds, the Dark value the
// brighter jewel tone for dark backgrounds. This is the fix for the
// "invisible on light mode" problem: never assume the background.
var (
	emerald     = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#10B981"}
	emeraldRule = lipgloss.AdaptiveColor{Light: "#A7D7C5", Dark: "#065F46"}
	rowAlt      = lipgloss.AdaptiveColor{Light: "#ECF5F0", Dark: "#123A2E"}

	gold = lipgloss.AdaptiveColor{Light: "#B8860B", Dark: "#D4AF37"}

	garnet = lipgloss.AdaptiveColor{Light: "#A01A38", Dark: "#E06C82"}

	text    = lipgloss.AdaptiveColor{Light: "#1C1B18", Dark: "#F5EFE0"}
	textDim = lipgloss.AdaptiveColor{Light: "#6B6455", Dark: "#A79E88"}

	// Ink is the dark foreground used on top of gold/emerald fills; it must
	// stay dark in both modes because the fill behind it is always bright.
	ink = lipgloss.AdaptiveColor{Light: "#161410", Dark: "#161410"}
	// onFill is the light foreground for dark fills, bright in both modes.
	onFill = lipgloss.AdaptiveColor{Light: "#FFF7EC", Dark: "#FFF7EC"}
	// statusFill is the solid strip behind the status-bar summary — a deep
	// emerald in both modes so cream text always has strong contrast.
	statusFill = lipgloss.AdaptiveColor{Light: "#065F46", Dark: "#0B3D2E"}
)

// Tier is the shared three-level scale used for coloring across the TUI —
// finding badges, panel status text, and threshold bars all resolve to one
// of these before picking a color, so "what color means what" lives in
// exactly one place instead of being re-decided by every switch statement
// that needs a traffic-light color.
type Tier int

const (
	TierGood Tier = iota
	TierWarn
	TierCrit
)

// tierPalette maps each Tier to its color. Changing what "warning" looks
// like anywhere in the UI means editing this one line.
var tierPalette = map[Tier]lipgloss.AdaptiveColor{
	TierGood: emerald,
	TierWarn: gold,
	TierCrit: garnet,
}

// Color returns this tier's color.
func (t Tier) Color() lipgloss.AdaptiveColor { return tierPalette[t] }

// severityTier maps a finding's Severity to its display Tier — used to
// find the single worst tier across a report's findings. Info reads as
// "good" here: it's informational, not a problem.
var severityTier = map[types.Severity]Tier{
	types.Crit: TierCrit,
	types.Warn: TierWarn,
	types.OK:   TierGood,
	types.Info: TierGood,
}

// severityStyle is one Severity's exact badge background/foreground pair.
type severityStyle struct {
	BG, FG lipgloss.TerminalColor
}

// severityStyles gives each Severity its exact badge colors. Distinct from
// tierPalette because Info uses a muted look, not the "good" green.
var severityStyles = map[types.Severity]severityStyle{
	types.Crit: {BG: garnet, FG: onFill},
	types.Warn: {BG: gold, FG: ink},
	types.OK:   {BG: emerald, FG: ink},
	types.Info: {BG: emeraldRule, FG: text},
}

var (
	styleDim  = lipgloss.NewStyle().Foreground(textDim)
	styleBody = lipgloss.NewStyle().Foreground(text)

	styleSectionHeading = lipgloss.NewStyle().Bold(true).Foreground(gold)
	styleKey            = lipgloss.NewStyle().Bold(true).Foreground(gold)

	panelBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(emeraldRule).
			Padding(0, 1)
)

// panel wraps content in the shared bordered-panel look. total is the
// exact number of terminal columns the panel occupies. lipgloss folds
// horizontal padding into the content width, so a rounded border adds 2
// columns total; the inner text area is therefore total-4.
func panel(total int, content string) string {
	if total < 6 {
		total = 6
	}
	return panelBorder.Width(total - 2).Render(content)
}

// innerWidth is the usable text width inside a panel of the given total.
func innerWidth(total int) int {
	w := total - 4
	if w < 1 {
		w = 1
	}
	return w
}

// panelTitled draws a panel with a heading line and a rule under it.
func panelTitled(total int, heading, content string) string {
	inner := innerWidth(total)
	rule := lipgloss.NewStyle().Foreground(emeraldRule).Render(strings.Repeat("─", inner))
	body := styleSectionHeading.Render(heading) + "\n" + rule + "\n" + content
	return panel(total, body)
}

// badge renders a fixed-width colored severity pill, e.g. " CRIT ".
func badge(sev types.Severity) string {
	st := severityStyles[sev]
	label := strings.ToUpper(string(sev))
	if len(label) > 4 {
		label = label[:4]
	}
	return lipgloss.NewStyle().
		Bold(true).
		Background(st.BG).
		Foreground(st.FG).
		Padding(0, 1).
		Render(fmt.Sprintf("%-4s", label))
}

// usageBar renders a filled/empty block bar colored by threshold — the
// same Tier scale the correlation rules effectively use, so the bar's
// color always agrees with the findings text beside it.
func usageBar(pct float64, width int) string {
	if width < 4 {
		width = 4
	}
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	full := lipgloss.NewStyle().Foreground(usageTier(pct).Color()).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(emeraldRule).Render(strings.Repeat("░", width-filled))
	return full + empty
}

const (
	usageBarWarnPct = 85.0
	usageBarCritPct = 95.0
)

// usageTier buckets a percent-used value into the shared Tier scale.
func usageTier(pct float64) Tier {
	switch {
	case pct >= usageBarCritPct:
		return TierCrit
	case pct >= usageBarWarnPct:
		return TierWarn
	default:
		return TierGood
	}
}

// statRow renders a "label   value" line with consistent label width.
func statRow(label, value string) string {
	return lipgloss.NewStyle().Foreground(textDim).Width(12).Render(label) + styleBody.Render(value)
}

// gradientText interpolates each rune's color between two hex colors.
// Used once, for the banner title. Takes explicit hex (not adaptive)
// because the caller passes mode-appropriate endpoints.
func gradientText(s, fromHex, toHex string) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}
	fr, fg, fb := hexToRGB(fromHex)
	tr, tg, tb := hexToRGB(toHex)

	var b strings.Builder
	for i, r := range runes {
		t := 0.0
		if n > 1 {
			t = float64(i) / float64(n-1)
		}
		hex := fmt.Sprintf("#%02x%02x%02x", lerp(fr, tr, t), lerp(fg, tg, t), lerp(fb, tb, t))
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(hex)).Render(string(r)))
	}
	return b.String()
}

// hexToRGB splits a "#rrggbb" string into its integer channels.
func hexToRGB(h string) (r, g, b int) {
	h = strings.TrimPrefix(h, "#")
	ri, _ := strconv.ParseInt(h[0:2], 16, 32)
	gi, _ := strconv.ParseInt(h[2:4], 16, 32)
	bi, _ := strconv.ParseInt(h[4:6], 16, 32)
	return int(ri), int(gi), int(bi)
}

// lerp linearly interpolates between a and b at position t (0..1).
func lerp(a, b int, t float64) int {
	return int(float64(a) + (float64(b)-float64(a))*t)
}
