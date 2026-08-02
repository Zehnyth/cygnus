// Package ui is the Bubble Tea layer. It only renders a *types.Report
// that services already built — it never reads /proc and never
// correlates data itself.
package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/zehnyth/cygnus/types"
)

// wideLayoutThreshold is the terminal width above which the memory and
// pressure panels sit side by side instead of stacked.
const wideLayoutThreshold = 78

// minVisibleRows keeps at least this many process rows on screen even in
// a very short terminal (the view may overflow, but stays usable).
const minVisibleRows = 5

// processTableChrome is the fixed number of lines the PROCESSES panel
// consumes beyond its data rows: title, rule, two borders, and the column
// header — plus one reserved unconditionally for the "showing X–Y of Z"
// line so the first scroll doesn't suddenly overflow by a row.
const processTableChrome = 6

// Model is the Bubble Tea model for the interactive report view: a
// Report plus the scroll/cursor state needed to browse its process table.
type Model struct {
	report *types.Report
	cursor int
	offset int // index of the first visible process row
	width  int
	height int

	// fetch re-runs the collection + correlation pipeline for a live
	// auto-refresh. nil for headless rendering (see Render), which never
	// calls Init/Update and so never needs it.
	fetch           func() (*types.Report, error)
	refreshInterval time.Duration
	paused          bool
	fetching        bool  // an async fetch is in flight — skip starting another
	lastErr         error // most recent refresh error, if any; cleared on success
}

// New creates a Model ready to run under tea.NewProgram. fetch re-runs the
// collection pipeline for auto-refresh; pass nil to disable refreshing.
func New(r *types.Report, fetch func() (*types.Report, error)) Model {
	return Model{report: r, fetch: fetch, refreshInterval: defaultRefreshInterval}
}

// Init satisfies tea.Model, kicking off the refresh timer when fetch is set.
func (m Model) Init() tea.Cmd {
	if m.fetch == nil {
		return nil
	}
	return tickCmd(m.refreshInterval)
}

// visibleRows is how many process rows fit given the current height. Used
// during Update() (scroll clamping) where there's no reason to render the
// full frame just to measure it — see chromeHeight.
func (m Model) visibleRows() int {
	n := m.height - m.chromeHeight()
	if n < minVisibleRows {
		n = minVisibleRows
	}
	if total := len(m.report.TopByPSS); n > total {
		n = total
	}
	return n
}

// topAndBottomSections renders everything except the process table's data
// rows, split into what comes before it and what comes after. Both
// chromeHeight (sizing, called from Update()) and layout (the real frame,
// called from View()) go through this exactly once each, so banner/
// topPanels/findings/footer/refreshStatus are never rendered twice for
// the same frame — they used to be: layout() built them directly AND
// chromeHeight() rebuilt them all over again just to measure a height,
// discarding the result. Same output, half the work, and one less place
// for the two renders to ever disagree.
func (m Model) topAndBottomSections(width int) (top, bottom string) {
	top = lipgloss.JoinVertical(lipgloss.Left,
		renderBanner(width),
		"",
		m.topPanels(width),
		renderFindings(m.report.Findings, width),
	)
	bottom = lipgloss.JoinVertical(lipgloss.Left,
		"",
		renderFooter(width),
		renderRefreshStatus(width, m.paused, m.lastErr),
	)
	return top, bottom
}

// chromeHeight measures every section except the process table's data rows
// by actually rendering them, rather than guessing at a fixed line count.
// Findings count (more rules fire at once exactly when the system is under
// real pressure), footer/finding text wrapping at narrow widths, and
// stacked-vs-side-by-side panel layout all change how tall the chrome
// actually is — a fixed constant reliably undercounts that and lets the
// rendered frame grow taller than the terminal. When that happens, the
// *top* of the frame (banner, the memory usage bar, the PSI
// stable/elevated/stalling status) is what scrolls out of view, since the
// process table and footer are rendered after it. Measuring instead of
// guessing is the fix.
func (m Model) chromeHeight() int {
	width := m.width
	if width < minWidth {
		width = minWidth
	}
	top, bottom := m.topAndBottomSections(width)
	return lipgloss.Height(top) + lipgloss.Height(bottom) + processTableChrome
}

// topPanels lays out the MEMORY and PRESSURE panels side by side above
// wideLayoutThreshold, stacked below it. In the side-by-side case their
// content is padded to equal line counts first, so both panels' bottom
// borders land on the same row instead of one trailing off short — MEMORY
// and PRESSURE's natural line counts don't match, and that mismatch is
// exactly what reads as "uneven" once they're side by side. Their top
// borders/headers are already guaranteed level: both go through
// panelTitled and are joined with lipgloss.JoinHorizontal(lipgloss.Top,
// ...), which aligns line 0 of each block regardless of how many lines
// follow.
func (m Model) topPanels(width int) string {
	verdictTier, verdictWord := worstVerdict(m.report)

	if width >= wideLayoutThreshold {
		half := width/2 - 1
		rest := width - half
		memLines := memoryPanelLines(m.report.Mem, half, verdictTier, verdictWord)
		presLines := pressurePanelLines(m.report.Pressure, m.report.Findings)

		target := len(memLines)
		if len(presLines) > target {
			target = len(presLines)
		}
		for len(memLines) < target {
			memLines = append(memLines, "")
		}
		for len(presLines) < target {
			presLines = append(presLines, "")
		}

		return lipgloss.JoinHorizontal(lipgloss.Top,
			panelTitled(half, "MEMORY", strings.Join(memLines, "\n")),
			panelTitled(rest, "PRESSURE", strings.Join(presLines, "\n")),
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		renderMemoryPanel(m.report.Mem, width, verdictTier, verdictWord),
		renderPressurePanel(m.report.Pressure, m.report.Findings, width),
	)
}

// Update satisfies tea.Model, handling window resizes, the scroll/quit
// keybindings (arrow keys, vim keys, page up/down, g/G, q), and the
// background auto-refresh cycle (tickMsg/reportMsg, p to pause/resume).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampScroll()
	case tickMsg:
		next := tickCmd(m.refreshInterval)
		if m.paused || m.fetching || m.fetch == nil {
			return m, next
		}
		m.fetching = true
		return m, tea.Batch(fetchCmd(m.fetch), next)
	case reportMsg:
		m.fetching = false
		if msg.err != nil {
			// Transient collection error: keep showing the last good
			// report, surface the error, try again next tick.
			m.lastErr = msg.err
			return m, nil
		}
		m.lastErr = nil
		m.applyReport(msg.report)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "p":
			m.paused = !m.paused
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			m.clampScroll()
		case "down", "j":
			if m.cursor < len(m.report.TopByPSS)-1 {
				m.cursor++
			}
			m.clampScroll()
		case "g", "home":
			m.cursor = 0
			m.clampScroll()
		case "G", "end":
			m.cursor = len(m.report.TopByPSS) - 1
			m.clampScroll()
		case "pgdown", "ctrl+d", "f":
			m.cursor += m.visibleRows()
			if m.cursor > len(m.report.TopByPSS)-1 {
				m.cursor = len(m.report.TopByPSS) - 1
			}
			m.clampScroll()
		case "pgup", "ctrl+u", "b":
			m.cursor -= m.visibleRows()
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.clampScroll()
		case "enter":
			// v2: drill into selected process (connections, open files, cgroup)
		}
	}
	return m, nil
}

// applyReport swaps in a freshly fetched report, keeping the cursor on the
// same process (by PID) across the refresh when it's still present. If that
// process is gone (exited), the cursor holds its old numeric row instead of
// jumping back to the top — "your place" is the spot on screen, not just
// the PID, when there's nothing better to track.
func (m *Model) applyReport(r *types.Report) {
	selectedPID := -1
	if m.cursor >= 0 && m.cursor < len(m.report.TopByPSS) {
		selectedPID = m.report.TopByPSS[m.cursor].PID
	}
	newCursor := m.cursor

	m.report = r

	if selectedPID != -1 {
		for i, p := range r.TopByPSS {
			if p.PID == selectedPID {
				newCursor = i
				break
			}
		}
	}
	switch {
	case newCursor >= len(r.TopByPSS):
		newCursor = len(r.TopByPSS) - 1
	case newCursor < 0:
		newCursor = 0
	}
	m.cursor = newCursor
	m.clampScroll()
}

// clampScroll keeps the cursor within the visible window, scrolling the
// window as needed so the selection is always on screen.
func (m *Model) clampScroll() {
	vis := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+vis {
		m.offset = m.cursor - vis + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// View satisfies tea.Model, delegating to layout.
func (m Model) View() string { return m.layout() }

// layout composes the full view for the model's current width. Kept
// separate from View so Render can produce output without a live terminal.
func (m Model) layout() string {
	width := m.width
	if width < minWidth {
		width = minWidth
	}

	top, bottom := m.topAndBottomSections(width)

	visRows := m.height - lipgloss.Height(top) - lipgloss.Height(bottom) - processTableChrome
	if visRows < minVisibleRows {
		visRows = minVisibleRows
	}
	if total := len(m.report.TopByPSS); visRows > total {
		visRows = total
	}

	procTable := renderProcessTable(m.report.TopByPSS, m.cursor, m.offset, visRows, width, m.report.Mem.TotalKB)
	return lipgloss.JoinVertical(lipgloss.Left, top, procTable, bottom)
}

// Render composes the view at a fixed width/height without a live
// terminal — the seam for scripted/headless output and future tests.
func Render(r *types.Report, width, height int) string {
	m := Model{report: r, width: width, height: height}
	m.clampScroll()
	return m.layout()
}
