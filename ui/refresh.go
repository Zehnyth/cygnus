package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/zehnyth/cygnus/types"
)

// defaultRefreshInterval is how often the TUI re-scrapes /proc when running
// and not paused.
const defaultRefreshInterval = 1 * time.Second

// tickMsg fires on the refresh schedule. It carries no data — Update reacts
// to it by (maybe) starting a fetch and always rescheduling the next tick.
type tickMsg time.Time

// reportMsg is the result of one async fetch: either a fresh Report or a
// transient error. Update must never let an error clobber what's already
// on screen — the whole point of refreshing is that it's allowed to fail
// quietly and try again next tick.
type reportMsg struct {
	report *types.Report
	err    error
}

// tickCmd schedules the next tickMsg after d without blocking the UI loop —
// tea.Tick runs on a timer goroutine owned by the Bubble Tea runtime, so
// Update/View keep handling keypresses while it waits.
func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// fetchCmd runs fetch on a goroutine (Bubble Tea's Cmd execution model), so
// a slow /proc scan never stalls keypress handling or the render loop. A
// panic inside fetch (e.g. an unexpected /proc format on some kernel) is
// recovered into a reportMsg error instead of crashing the whole TUI.
func fetchCmd(fetch func() (*types.Report, error)) tea.Cmd {
	return func() (msg tea.Msg) {
		defer func() {
			if rec := recover(); rec != nil {
				msg = reportMsg{err: fmt.Errorf("panic during refresh: %v", rec)}
			}
		}()
		r, err := fetch()
		return reportMsg{report: r, err: err}
	}
}
