package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/zehnyth/cygnus/types"
	"github.com/zehnyth/cygnus/utils"
)

// renderText writes the static (non-TUI, non-JSON) report view to w.
// Takes an io.Writer rather than assuming os.Stdout so it can be
// exercised in tests with a bytes.Buffer.
func renderText(w io.Writer, r *types.Report) {
	_, _ = fmt.Fprintf(w, "MEMORY  total %.1f GB · available %.1f GB (%.1f%%) · cache+buffers %.1f GB · slab %.1f GB · dirty %.0f MB\n",
		utils.GB(r.Mem.TotalKB), utils.GB(r.Mem.AvailableKB), r.Mem.AvailablePct,
		utils.GB(r.Mem.BuffersKB+r.Mem.CachedKB), utils.GB(r.Mem.SReclaimKB+r.Mem.SUnreclaimKB),
		float64(r.Mem.DirtyKB)/1024)

	if r.Pressure.Supported {
		_, _ = fmt.Fprintf(w, "PSI     some avg10=%.2f avg60=%.2f · full avg10=%.2f avg60=%.2f\n",
			r.Pressure.SomeAvg10, r.Pressure.SomeAvg60, r.Pressure.FullAvg10, r.Pressure.FullAvg60)
	}
	if r.MatchName != "" {
		_, _ = fmt.Fprintf(w, "MATCH   %q → %d processes\n", r.MatchName, r.MatchHits)
	}

	_, _ = fmt.Fprintln(w, "\nFINDINGS")
	for _, f := range r.Findings {
		_, _ = fmt.Fprintf(w, "  [%s] %s\n", f.Severity, f.Text)
	}

	_, _ = fmt.Fprintln(w, "\nTOP BY PSS")
	tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "  PID\tUSER\tPSS\tRSS\tSWAP\tTHR\tFDS\tCOMMAND")
	for _, p := range r.TopByPSS {
		_, _ = fmt.Fprintf(tw, "  %d\t%s\t%.0f MB\t%.0f MB\t%.0f MB\t%d\t%s\t%s\n",
			p.PID, p.User, utils.MB(p.PssKB), utils.MB(p.RssKB), utils.MB(p.SwapKB),
			p.Threads, utils.FDs(p.FDs), p.Comm)
	}
	_ = tw.Flush()
}
