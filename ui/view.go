package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/zehnyth/cygnus/types"
	"github.com/zehnyth/cygnus/utils"
)

const minWidth = 64 // fallback before the first WindowSizeMsg arrives

// bannerHexes returns gradient endpoints tuned to the background mode.
// lipgloss.HasDarkBackground drives the choice so the title stays vivid
// on dark and readable on light.
func bannerHexes() (from, to string) {
	if lipgloss.HasDarkBackground() {
		return "#10B981", "#F2C14E"
	}
	return "#047857", "#A97400"
}

// renderBanner draws the "✦ CYGNUS ✦" title line with a gradient and the
// "PSS-accurate" tag right-aligned, degrading to left-aligned when narrow.
func renderBanner(width int) string {
	from, to := bannerHexes()
	title := gradientText("✦ CYGNUS ✦", from, to)
	sub := styleDim.Render("host memory triage")
	left := title + "  " + sub

	right := styleDim.Render("PSS-accurate")
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		// Too narrow for the right-side tag — pad left to full width instead.
		pad := max(0, width-lipgloss.Width(left))
		return left + strings.Repeat(" ", pad)
	}
	return left + strings.Repeat(" ", gap) + right
}

// statusBar renders the top strip: a colored verdict segment plus a
// summary, styled like the reference's STATUS bar. Both segments use
// fills with fixed-contrast foregrounds so the bar is legible on any
// terminal background — this is deliberately NOT adaptive text on a
// pale fill, which is what made light mode unreadable.
func statusBar(r *types.Report, width int) string {
	label, segBg := worstVerdict(r)

	seg := lipgloss.NewStyle().
		Bold(true).
		Foreground(ink).
		Background(segBg).
		Padding(0, 1).
		Render(label)

	crit, warn := countFindings(r.Findings)
	summary := fmt.Sprintf("  %d critical · %d warning · %.0f%% mem available",
		crit, warn, r.Mem.AvailablePct)

	rest := max(0, width-lipgloss.Width(seg))
	body := lipgloss.NewStyle().
		Foreground(onFill).
		Background(statusFill).
		Width(rest).
		Render(summary)

	return seg + body
}

// verdictLabel names each Tier for the status bar's headline word.
var verdictLabel = map[Tier]string{
	TierGood: "HEALTHY",
	TierWarn: "WARNING",
	TierCrit: "CRITICAL",
}

// worstVerdict returns the status-bar headline label and color for the
// single worst Tier across all of a report's findings.
func worstVerdict(r *types.Report) (string, lipgloss.TerminalColor) {
	worst := TierGood
	for _, f := range r.Findings {
		if t := severityTier[f.Severity]; t > worst {
			worst = t
		}
	}
	return verdictLabel[worst], worst.Color()
}

// countFindings tallies Crit and Warn findings for the status-bar summary.
func countFindings(fs []types.Finding) (crit, warn int) {
	for _, f := range fs {
		switch f.Severity {
		case types.Crit:
			crit++
		case types.Warn:
			warn++
		}
	}
	return
}

// renderMemoryPanel draws the bordered MEMORY panel: a usage bar plus
// total/available/cache/slab/dirty stat rows.
func renderMemoryPanel(m *types.MemInfo, width int) string {
	usedPct := 100 - m.AvailablePct
	inner := innerWidth(width)

	bar := usageBar(usedPct, inner-10) + styleBody.Render(fmt.Sprintf(" %.0f%% used", usedPct))
	lines := []string{
		bar,
		"",
		statRow("Total", fmt.Sprintf("%.1f GB", utils.GB(m.TotalKB))),
		statRow("Available", fmt.Sprintf("%.1f GB (%.0f%%)", utils.GB(m.AvailableKB), m.AvailablePct)),
		statRow("Cache+Buf", fmt.Sprintf("%.1f GB", utils.GB(m.BuffersKB+m.CachedKB))),
		statRow("Slab", fmt.Sprintf("%.1f GB", utils.GB(m.SReclaimKB+m.SUnreclaimKB))),
		statRow("Dirty", fmt.Sprintf("%.0f MB", float64(m.DirtyKB)/1024)),
	}
	return panelTitled(width, "MEMORY", strings.Join(lines, "\n"))
}

// renderPressurePanel draws the bordered PRESSURE panel: a stable/elevated/
// stalling status word plus the four PSI avg stat rows, or a fallback
// message when PSI isn't supported by the running kernel.
func renderPressurePanel(p *types.PSI, width int) string {
	if !p.Supported {
		return panelTitled(width, "PRESSURE", styleDim.Render("PSI not available\non this kernel."))
	}

	tier := psiTier(p)
	status := lipgloss.NewStyle().Bold(true).Foreground(tier.Color()).Render("● " + psiStatusText[tier])

	lines := []string{
		status,
		"",
		statRow("Some avg10", fmt.Sprintf("%.2f", p.SomeAvg10)),
		statRow("Some avg60", fmt.Sprintf("%.2f", p.SomeAvg60)),
		statRow("Full avg10", fmt.Sprintf("%.2f", p.FullAvg10)),
		statRow("Full avg60", fmt.Sprintf("%.2f", p.FullAvg60)),
	}
	return panelTitled(width, "PRESSURE", strings.Join(lines, "\n"))
}

// PSI display thresholds are independent of the services rule-engine
// thresholds (services.Config) — these color a live status word in the
// TUI panel, not a Finding, so they stay local visual constants.
const (
	psiSomeWarnAvg60 = 5.0
	psiFullCritAvg60 = 1.0
)

var psiStatusText = map[Tier]string{
	TierGood: "stable",
	TierWarn: "elevated",
	TierCrit: "stalling",
}

// psiTier buckets a PSI reading into the shared Tier scale for the
// pressure panel's status word.
func psiTier(p *types.PSI) Tier {
	switch {
	case p.FullAvg60 > psiFullCritAvg60:
		return TierCrit
	case p.SomeAvg60 > psiSomeWarnAvg60:
		return TierWarn
	default:
		return TierGood
	}
}

// renderFindings draws the bordered FINDINGS panel: one severity-badged
// line per finding.
func renderFindings(findings []types.Finding, width int) string {
	inner := innerWidth(width)
	rows := make([]string, 0, len(findings))
	for _, f := range findings {
		line := badge(f.Severity) + " " + styleBody.Render(f.Text)
		rows = append(rows, lipgloss.NewStyle().Width(inner).Render(line))
	}
	return panelTitled(width, "FINDINGS", strings.Join(rows, "\n"))
}

// Column widths for the process table (content only; a single space gap
// is added between columns by the renderer, so columns never touch even
// when a value fills its cell). PID is 8 wide to fit modern large PIDs
// (kernel.pid_max is commonly 4194304 = 7 digits, plus headroom).
var procCols = []int{8, 12, 8, 8, 7, 5, 5}

const colGap = " " // separation guaranteed between every column

// renderProcessTable draws the bordered PROCESSES panel: a header row plus
// the [offset, offset+maxRows) window of procs, highlighting cursor and
// alternating row shading, with a "showing X–Y of Z" footer when scrolled.
func renderProcessTable(procs []types.Proc, cursor, offset, maxRows, width int) string {
	header := joinCols(
		hCell(procCols[0], lipgloss.Left, "PID"),
		hCell(procCols[1], lipgloss.Left, "USER"),
		hCell(procCols[2], lipgloss.Right, "PSS"),
		hCell(procCols[3], lipgloss.Right, "RSS"),
		hCell(procCols[4], lipgloss.Right, "SWAP"),
		hCell(procCols[5], lipgloss.Right, "THR"),
		hCell(procCols[6], lipgloss.Right, "FDS"),
		lipgloss.NewStyle().Bold(true).Foreground(gold).Render("COMMAND"),
	)

	// Clamp the visible window to [offset, offset+maxRows).
	end := offset + maxRows
	if end > len(procs) {
		end = len(procs)
	}
	if offset < 0 {
		offset = 0
	}

	rows := []string{header}
	for i := offset; i < end; i++ {
		rows = append(rows, renderProcRow(procs[i], i == cursor, i%2 == 1, innerWidth(width)))
	}

	// Footer line showing position when the list is scrolled/truncated.
	if len(procs) > maxRows {
		pos := fmt.Sprintf("showing %d–%d of %d — ↑/↓ to scroll", offset+1, end, len(procs))
		rows = append(rows, lipgloss.NewStyle().Foreground(textDim).Render(pos))
	}

	title := fmt.Sprintf("PROCESSES · by PSS · %d total", len(procs))
	return panelTitled(width, title, strings.Join(rows, "\n"))
}

// joinCols concatenates rendered cells with a single-space gap between each.
func joinCols(cells ...string) string {
	return strings.Join(cells, colGap)
}

// hCell renders one bold gold-on-default table header cell.
func hCell(w int, align lipgloss.Position, s string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(gold).Width(w).Align(align).Render(s)
}

// renderProcRow renders one process's table row, styled for selection
// (cursor row) or alternating-row shading, with COMMAND truncated to fit.
func renderProcRow(p types.Proc, selected, alt bool, rowWidth int) string {
	fg := text
	var bg lipgloss.TerminalColor
	switch {
	case selected:
		fg, bg = ink, gold
	case alt:
		bg = rowAlt
	}

	cell := lipgloss.NewStyle().Foreground(fg)
	if bg != nil {
		cell = cell.Background(bg)
	}
	pss := cell.Bold(true) // PSS is the number worth trusting — emphasize it

	// Fixed columns plus 6 single-space gaps between the 7 fixed columns
	// and 1 gap before COMMAND = 7 gap columns total.
	fixed := procCols[0] + procCols[1] + procCols[2] + procCols[3] + procCols[4] + procCols[5] + procCols[6]
	cmdWidth := rowWidth - fixed - 7
	if cmdWidth < 4 {
		cmdWidth = 4
	}

	row := joinCols(
		cell.Width(procCols[0]).Render(fmt.Sprintf("%d", p.PID)),
		cell.Width(procCols[1]).Render(truncate(p.User, procCols[1])),
		pss.Width(procCols[2]).Align(lipgloss.Right).Render(fmt.Sprintf("%.0fM", utils.MB(p.PssKB))),
		cell.Width(procCols[3]).Align(lipgloss.Right).Render(fmt.Sprintf("%.0fM", utils.MB(p.RssKB))),
		cell.Width(procCols[4]).Align(lipgloss.Right).Render(fmt.Sprintf("%.0fM", utils.MB(p.SwapKB))),
		cell.Width(procCols[5]).Align(lipgloss.Right).Render(fmt.Sprintf("%d", p.Threads)),
		cell.Width(procCols[6]).Align(lipgloss.Right).Render(utils.FDs(p.FDs)),
		cell.Width(cmdWidth).Render(truncate(p.Comm, cmdWidth)),
	)
	return row
}

// renderFooter draws the centered keybinding hint line.
func renderFooter(width int) string {
	hint := func(key, desc string) string {
		return styleKey.Render(key) + styleDim.Render(" "+desc)
	}
	sep := styleDim.Render("   •   ")
	keys := strings.Join([]string{
		hint("↑/↓", "scroll"),
		hint("g/G", "top/bottom"),
		hint("enter", "inspect"),
		hint("q", "quit"),
	}, sep)
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(keys)
}

// truncate shortens s to at most max runes, appending "…" if it was cut.
func truncate(s string, max int) string {
	r := []rune(s)
	if max < 1 || len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

// max returns the larger of a and b.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
