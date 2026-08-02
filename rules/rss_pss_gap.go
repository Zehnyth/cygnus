package rules

import (
	"fmt"

	"github.com/zehnyth/cygnus/types"
)

func init() {
	Register(types.Rule{
		Name:        "rss-pss-gap",
		Description: "Quantifies how much a naive summed-RSS view overstates real usage vs PSS, or reports degraded PSS collection.",
		Priority:    50,
		Check:       rssPssGap,
	})
}

// rssPssGap reports degraded PSS collection (non-root run) if it occurred,
// otherwise fires when summed RSS overstates summed PSS beyond the
// configured ratio — the double-counted-shared-pages problem this whole
// tool exists to correct.
func rssPssGap(cfg types.Config, r *types.Report, procs []types.Proc) *types.Finding {
	if r.Degraded {
		return &types.Finding{Severity: types.Info, Text: fmt.Sprintf(
			"Per-process memory figures for %d of %d processes are estimates (RSS), not exact — cygnus isn't running as root. Run with sudo for accurate numbers.",
			r.DegradedCount, r.TotalProcs)}
	}
	var rss, pss uint64
	for _, p := range procs {
		rss += p.RssKB
		pss += p.PssKB
	}
	if pss > 0 && float64(rss)/float64(pss) > cfg.RSSPSSGapRatio {
		return &types.Finding{Severity: types.Info, Text: fmt.Sprintf(
			"Summed RSS across all processes (%.1f GB) overstates real usage by %.0f%% — the accurate figure (PSS) is %.1f GB. The gap is shared memory counted more than once. Trust PSS.",
			float64(rss)/1048576, 100*(float64(rss)/float64(pss)-1), float64(pss)/1048576)}
	}
	return nil
}
