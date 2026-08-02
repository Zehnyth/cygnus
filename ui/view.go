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

// renderBanner draws the "✦ C Y G N U S ✦" wordmark, centered across the
// full width, with the "PSS-accurate" tag alongside it. A terminal cell
// grid has no font-size control, so "larger" means letter-spacing the
// title wider rather than making it taller — that reads as a bigger
// wordmark without costing the extra rows a multi-line ASCII banner would
// (rows are scarce; see chromeHeight). There's no separate status strip
// below this anymore — the overall verdict now lives at the top of the
// MEMORY panel instead, where "healthy/warning/critical" sits next to the
// numbers that explain it, rather than floating above them repeating a
// summary.
func renderBanner(width int) string {
	from, to := bannerHexes()
	title := gradientText("✦  CYGNUS  ✦", from, to)
	content := title + "   "
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(content)
}

// verdictLabel names each Tier for the MEMORY panel's headline word.
var verdictLabel = map[Tier]string{
	TierGood: "healthy",
	TierWarn: "warning",
	TierCrit: "critical",
}

// verdictDescription gives each verdict Tier a one-line explanation. The
// verdict is the worst finding across the *whole* report, not just
// memory, so this deliberately stays generic rather than claiming
// something specifically about memory — see the FINDINGS panel for what
// actually fired.
var verdictDescription = map[Tier]string{
	TierGood: "No issues detected.",
	TierWarn: "One or more warnings — see findings below.",
	TierCrit: "Critical issue detected — see findings below.",
}

// worstVerdict returns the single worst Tier across a report's findings
// and its label, for use with renderStatusLine.
func worstVerdict(r *types.Report) (Tier, string) {
	worst := TierGood
	for _, f := range r.Findings {
		if t := severityTier[f.Severity]; t > worst {
			worst = t
		}
	}
	return worst, verdictLabel[worst]
}

// renderStatusLine renders a panel's headline status: a colored dot, an
// uppercase label, and a one-line description beneath it. MEMORY and
// PRESSURE both build their status through this exact component instead
// of two parallel one-off layouts — dot and label always share the same
// color source, so they can't drift out of sync, and the two panels'
// status presentation is structurally identical.
func renderStatusLine(tier Tier, label, description string) []string {
	return []string{
		lipgloss.NewStyle().Bold(true).Foreground(tier.Color()).Render("● " + strings.ToUpper(label)),
		styleDim.Render(description),
	}
}

// renderMemoryPanel draws the bordered MEMORY panel: the overall verdict,
// total RAM, one bar for the used/free split, then plain-value context
// rows for what the rest is doing.
func renderMemoryPanel(m *types.MemInfo, width int, verdictTier Tier, verdictWord string) string {
	lines := memoryPanelLines(m, width, verdictTier, verdictWord)
	return panelTitled(width, "MEMORY", strings.Join(lines, "\n"))
}

// memoryPanelLines is MEMORY's content, one slice element per rendered
// line. Split out from renderMemoryPanel so topPanels can measure and pad
// it to match PRESSURE's line count before either gets its border drawn —
// that's what keeps the two side-by-side panels' bottom borders aligned
// instead of one trailing off a line short of the other.
//
// There is exactly ONE bar here, for Used vs. Free — that's the only pair
// that's genuinely mutually exclusive and adds up to Total. Free gets its
// own row directly under the bar (not text floating at the end of the bar
// line) so it reads as the headline number it is, before the secondary
// breakdown. Page Cache and Buffers are reported as two separate fields by
// the kernel and are shown separately here for the same reason — merging
// them under one "Cache+Buf" label hid that they're different things.
// Slab and Dirty are context, not separate slices of Total: Slab is mostly
// already counted inside Free (the kernel reclaims it on demand, same idea
// as the cache-not-pressure finding), and Dirty is a subset of Page Cache,
// not a sibling of it. Every context row — Total included — goes through
// the same three-column memInfoRow (label, value, note) so nothing is
// ragged: a blank note column reads as an omission, so every row gets one.
func memoryPanelLines(m *types.MemInfo, width int, verdictTier Tier, verdictWord string) []string {
	usedPct := 100 - m.AvailablePct
	barWidth := innerWidth(width) - 12
	if barWidth < 6 {
		barWidth = 6
	}

	usageLine := drawBar(usedPct, barWidth, usageTier(usedPct).Color()) +
		styleBody.Render(fmt.Sprintf(" %.0f%% used", usedPct))

	swapUsedKB := uint64(0)
	if m.SwapTotalKB > m.SwapFreeKB {
		swapUsedKB = m.SwapTotalKB - m.SwapFreeKB
	}
	swapNote := fmt.Sprintf("of %.1f GB total", utils.GB(m.SwapTotalKB))
	if m.SwapTotalKB == 0 {
		swapNote = "no swap configured"
	}

	lines := renderStatusLine(verdictTier, verdictWord, verdictDescription[verdictTier])
	return append(lines,
		"",
		memInfoRow("Total", fmt.Sprintf("%.1f GB", utils.GB(m.TotalKB)), "physical RAM"),
		usageLine,
		memInfoRow("Free", fmt.Sprintf("%.1f GB", utils.GB(m.AvailableKB)), "available now"),
		"",
		memInfoRow("Page Cache", fmt.Sprintf("%.1f GB", utils.GB(m.CachedKB)), "reclaimable"),
		memInfoRow("Buffers", fmt.Sprintf("%.1f GB", utils.GB(m.BuffersKB)), "reclaimable"),
		memInfoRow("Slab", fmt.Sprintf("%.1f GB", utils.GB(m.SReclaimKB+m.SUnreclaimKB)), "kernel memory"),
		memInfoRow("Dirty", fmt.Sprintf("%.0f MB", float64(m.DirtyKB)/1024), "pending write"),
		"",
		memInfoRow("Swap Used", fmt.Sprintf("%.1f GB", utils.GB(swapUsedKB)), swapNote),
	)
}

// memInfoRow renders one MEMORY context row: a label, a plain value (no
// bar — see memoryPanelLines for why), and a short tag explaining what it
// is. Every row uses this, Total included, so the column is never ragged.
func memInfoRow(label, value, note string) string {
	return lipgloss.NewStyle().Foreground(textDim).Width(11).Render(label) +
		styleBody.Width(9).Render(value) +
		styleDim.Render(note)
}

// renderPressurePanel draws the bordered PRESSURE panel: a stable/elevated/
// stalling status word with a one-line plain-English explanation, plus a
// clean aligned Some/Full × avg10/avg60 grid. findings is the report's
// full findings list — see pressureTier for why the headline is derived
// from it rather than from p directly.
func renderPressurePanel(p *types.PSI, findings []types.Finding, width int) string {
	return panelTitled(width, "PRESSURE", strings.Join(pressurePanelLines(p, findings), "\n"))
}

// pressurePanelLines is PRESSURE's content, one slice element per rendered
// line — see memoryPanelLines for why this is split out.
func pressurePanelLines(p *types.PSI, findings []types.Finding) []string {
	if !p.Supported {
		return []string{
			styleDim.Render("PSI not available"),
			styleDim.Render("on this kernel."),
		}
	}

	tier := pressureTier(findings)
	lines := renderStatusLine(tier, psiStatusText[tier], pressureDescription[tier])
	lines = append(lines, "")
	return append(lines, pressureStatGrid(p)...)
}

// pressureTier derives PRESSURE's headline severity from the psi-stalls
// finding the rule engine already produced — using services.Config's
// thresholds — instead of recomputing it from raw PSI values against a
// second, independent set of thresholds. Two threshold systems judging the
// same signal is exactly what let MEMORY's verdict show "HEALTHY" while
// PRESSURE's own headline independently showed "WARNING"/"CRITICAL" for
// the same report: a value could cross the UI's local threshold without
// crossing services.Config's, or vice versa. Reading the actual fired
// finding makes the two panels structurally unable to disagree.
func pressureTier(findings []types.Finding) Tier {
	for _, f := range findings {
		if f.Rule == "psi-stalls" {
			return severityTier[f.Severity]
		}
	}
	return TierGood
}

var psiStatusText = map[Tier]string{
	TierGood: "stable",
	TierWarn: "elevated",
	TierCrit: "stalling",
}

// pressureDescription gives each tier a one-line, jargon-free explanation
// of what the status word actually means for the system right now.
var pressureDescription = map[Tier]string{
	TierGood: "Tasks aren't waiting on memory.",
	TierWarn: "Tasks are occasionally waiting on memory.",
	TierCrit: "Tasks are stuck waiting on memory right now.",
}

// pressureStatGrid's own thresholds are independent of the psi-stalls
// rule's Config thresholds on purpose: this grid colors individual Some/
// Full *numbers* for at-a-glance nuance, it never renders a status word
// like "HEALTHY"/"CRITICAL" — so it can't reproduce the panel-vs-panel
// contradiction pressureTier exists to prevent. Some and Full use
// different scales because "full" (every task blocked) is inherently
// worse than "some" at the same percentage.
const (
	pressureGridSomeWarn = 5.0
	pressureGridSomeCrit = 20.0
	pressureGridFullWarn = 0.5
	pressureGridFullCrit = 1.0
)

// someTier buckets a "some" PSI reading (at least one task stalled) into
// the shared Tier scale, for pressureStatGrid's row coloring only.
func someTier(avg60 float64) Tier {
	switch {
	case avg60 > pressureGridSomeCrit:
		return TierCrit
	case avg60 > pressureGridSomeWarn:
		return TierWarn
	default:
		return TierGood
	}
}

// fullTier buckets a "full" PSI reading (every task stalled) into the
// shared Tier scale, for pressureStatGrid's row coloring only.
func fullTier(avg60 float64) Tier {
	switch {
	case avg60 > pressureGridFullCrit:
		return TierCrit
	case avg60 > pressureGridFullWarn:
		return TierWarn
	default:
		return TierGood
	}
}

// pressureStatColWidth/pressureStatLabelWidth size pressureStatGrid's
// columns — kept as named constants so the header row and data rows can't
// drift out of alignment with each other.
const (
	pressureStatLabelWidth = 6
	pressureStatColWidth   = 8
)

// pressureStatGrid renders Some/Full × avg10/avg60 as a clean aligned
// grid: a label column plus two right-aligned value columns, matching the
// label/value alignment discipline used in MEMORY's context rows. avg60 is
// bold and severity-colored (it's the number worth trusting for "is this
// ongoing"); avg10 is a plain supporting figure. Replaces an earlier
// version with a tiny inline bar and a parenthetical secondary stat
// crammed onto one line — an aligned table reads cleaner at this size
// than a bar that's too narrow to carry real proportional meaning.
func pressureStatGrid(p *types.PSI) []string {
	header := lipgloss.NewStyle().Width(pressureStatLabelWidth).Render("") +
		lipgloss.NewStyle().Foreground(textDim).Width(pressureStatColWidth).Align(lipgloss.Right).Render("avg10") +
		lipgloss.NewStyle().Foreground(textDim).Width(pressureStatColWidth).Align(lipgloss.Right).Render("avg60")

	return []string{
		header,
		pressureStatRow("Some", p.SomeAvg10, p.SomeAvg60, someTier(p.SomeAvg60)),
		pressureStatRow("Full", p.FullAvg10, p.FullAvg60, fullTier(p.FullAvg60)),
	}
}

// pressureStatRow renders one row of pressureStatGrid.
func pressureStatRow(label string, avg10, avg60 float64, tier Tier) string {
	return lipgloss.NewStyle().Foreground(textDim).Width(pressureStatLabelWidth).Render(label) +
		styleDim.Width(pressureStatColWidth).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f%%", avg10)) +
		lipgloss.NewStyle().Bold(true).Foreground(tier.Color()).Width(pressureStatColWidth).Align(lipgloss.Right).Render(fmt.Sprintf("%.1f%%", avg60))
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
// (kernel.pid_max is commonly 4194304 = 7 digits, plus headroom). Every
// column is center-aligned, header and data alike, except COMMAND, which
// stays left as free-form text — centering a long, variable-length string
// reads worse than a ragged-right list, unlike the short fixed-shape
// values in every other column. %MEM sits right after PSS since it's the
// same figure reframed as a share of total system memory, not a new,
// unrelated measurement.
var procCols = []int{8, 10, 9, 7, 9, 9, 8, 6}

const colGap = " " // separation guaranteed between every column

// renderProcessTable draws the bordered PROCESSES panel: a header row plus
// the [offset, offset+maxRows) window of procs, highlighting cursor and
// alternating row shading, with a "showing X–Y of Z" footer when scrolled.
// totalKB (system total memory) is needed to compute each row's %MEM.
func renderProcessTable(procs []types.Proc, cursor, offset, maxRows, width int, totalKB uint64) string {
	header := joinCols(
		hCell(procCols[0], lipgloss.Center, "PID"),
		hCell(procCols[1], lipgloss.Center, "USER"),
		hCell(procCols[2], lipgloss.Center, "PSS"),
		hCell(procCols[3], lipgloss.Center, "%MEM"),
		hCell(procCols[4], lipgloss.Center, "RSS"),
		hCell(procCols[5], lipgloss.Center, "SWAP"),
		hCell(procCols[6], lipgloss.Center, "THREADS"),
		hCell(procCols[7], lipgloss.Center, "FDS"),
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
		rows = append(rows, renderProcRow(procs[i], i == cursor, i%2 == 1, innerWidth(width), totalKB))
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
// totalKB drives %MEM; a PSS value that's actually a RSS fallback
// (PssOK=false, not root) is marked with a dim "~" so degraded rows are
// visible individually, not just as an aggregate count in a finding.
func renderProcRow(p types.Proc, selected, alt bool, rowWidth int, totalKB uint64) string {
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

	pssText := fmt.Sprintf("%.0fM", utils.MB(p.PssKB))
	if !p.PssOK {
		pssText += "~" // estimated from RSS, not real PSS — see rss-pss-gap finding
		pss = cell.Foreground(textDim)
	}

	pctMem := 0.0
	if totalKB > 0 {
		pctMem = 100 * float64(p.PssKB) / float64(totalKB)
	}

	// Fixed columns plus 7 single-space gaps between the 8 fixed columns
	// and 1 gap before COMMAND = 8 gap columns total.
	fixed := procCols[0] + procCols[1] + procCols[2] + procCols[3] + procCols[4] + procCols[5] + procCols[6] + procCols[7]
	cmdWidth := rowWidth - fixed - 8
	if cmdWidth < 4 {
		cmdWidth = 4
	}

	row := joinCols(
		cell.Width(procCols[0]).Align(lipgloss.Center).Render(fmt.Sprintf("%d", p.PID)),
		cell.Width(procCols[1]).Align(lipgloss.Center).Render(truncate(p.User, procCols[1])),
		pss.Width(procCols[2]).Align(lipgloss.Center).Render(pssText),
		cell.Width(procCols[3]).Align(lipgloss.Center).Render(fmt.Sprintf("%.1f%%", pctMem)),
		cell.Width(procCols[4]).Align(lipgloss.Center).Render(fmt.Sprintf("%.0fM", utils.MB(p.RssKB))),
		cell.Width(procCols[5]).Align(lipgloss.Center).Render(fmt.Sprintf("%.0fM", utils.MB(p.SwapKB))),
		cell.Width(procCols[6]).Align(lipgloss.Center).Render(fmt.Sprintf("%d", p.Threads)),
		cell.Width(procCols[7]).Align(lipgloss.Center).Render(utils.FDs(p.FDs)),
		cell.Width(cmdWidth).Render(truncate(p.Comm, cmdWidth)),
	)
	return row
}

// renderFooter draws the keybinding hints centered in the space left of
// the "Developed by Zehnyth" credit, which is pinned to the far right. On
// a terminal too narrow to fit both, the credit drops so the hints — the
// functional part — never get cut off.
func renderFooter(width int) string {
	hint := func(key, desc string) string {
		return styleKey.Render(key) + styleDim.Render(" "+desc)
	}
	sep := styleDim.Render("   •   ")
	keys := strings.Join([]string{
		hint("↑/↓", "scroll"),
		hint("g/G", "top/bottom"),
		hint("p", "pause"),
		hint("enter", "inspect"),
		hint("q", "quit"),
	}, sep)

	credit := styleDim.Render("Developed by Zehnyth")
	mainWidth := width - lipgloss.Width(credit) - 2
	if mainWidth < lipgloss.Width(keys) {
		return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(keys)
	}
	return lipgloss.NewStyle().Width(mainWidth).Align(lipgloss.Center).Render(keys) + "  " + credit
}

// renderRefreshStatus draws a single centered status line for the
// auto-refresh cycle: a paused notice takes priority over a stale error
// (once resumed and healthy again, both clear and this renders blank).
func renderRefreshStatus(width int, paused bool, lastErr error) string {
	switch {
	case paused:
		return lipgloss.NewStyle().Foreground(gold).Width(width).Align(lipgloss.Center).
			Render("⏸ auto-refresh paused — press p to resume")
	case lastErr != nil:
		msg := truncate(fmt.Sprintf("⚠ refresh failed, showing last good data: %v", lastErr), width)
		return lipgloss.NewStyle().Foreground(garnet).Width(width).Align(lipgloss.Center).Render(msg)
	default:
		return ""
	}
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
