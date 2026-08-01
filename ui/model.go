// Package ui is the Bubble Tea layer. It only renders a *types.Report
// that services already built — it never reads /proc and never
// correlates data itself.
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/zehnyth/cygnus/types"
)

// wideLayoutThreshold is the terminal width above which the memory and
// pressure panels sit side by side instead of stacked.
const wideLayoutThreshold = 78

// chromeRows is the number of terminal lines consumed by everything that
// isn't a process row: banner, status bar, the two stat panels, findings,
// borders, footer, and spacing. Used to size the scrolling process window
// so the whole view fits without the terminal itself scrolling.
const chromeRows = 24

// minVisibleRows keeps at least this many process rows on screen even in
// a very short terminal (the view may overflow, but stays usable).
const minVisibleRows = 5

// Model is the Bubble Tea model for the interactive report view: a
// Report plus the scroll/cursor state needed to browse its process table.
type Model struct {
	report *types.Report
	cursor int
	offset int // index of the first visible process row
	width  int
	height int
}

// New creates a Model ready to run under tea.NewProgram.
func New(r *types.Report) Model { return Model{report: r} }

// Init satisfies tea.Model; cygnus has no startup command to run.
func (m Model) Init() tea.Cmd { return nil }

// visibleRows is how many process rows fit given the current height.
func (m Model) visibleRows() int {
	n := m.height - chromeRows
	if n < minVisibleRows {
		n = minVisibleRows
	}
	if total := len(m.report.TopByPSS); n > total {
		n = total
	}
	return n
}

// Update satisfies tea.Model, handling window resizes and the scroll/quit
// keybindings (arrow keys, vim keys, page up/down, g/G, q).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.clampScroll()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
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

	var top string
	if width >= wideLayoutThreshold {
		half := width/2 - 1
		top = lipgloss.JoinHorizontal(lipgloss.Top,
			renderMemoryPanel(m.report.Mem, half),
			renderPressurePanel(m.report.Pressure, width-half),
		)
	} else {
		top = lipgloss.JoinVertical(lipgloss.Left,
			renderMemoryPanel(m.report.Mem, width),
			renderPressurePanel(m.report.Pressure, width),
		)
	}

	sections := []string{
		renderBanner(width),
		statusBar(m.report, width),
		"",
		top,
		renderFindings(m.report.Findings, width),
		renderProcessTable(m.report.TopByPSS, m.cursor, m.offset, m.visibleRows(), width),
		"",
		renderFooter(width),
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// Render composes the view at a fixed width/height without a live
// terminal — the seam for scripted/headless output and future tests.
func Render(r *types.Report, width, height int) string {
	m := Model{report: r, width: width, height: height}
	m.clampScroll()
	return m.layout()
}
