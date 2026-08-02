package rules

import (
	"fmt"

	"github.com/zehnyth/cygnus/types"
)

func init() {
	Register(types.Rule{
		Name:        "psi-stalls",
		Description: "Flags sustained PSI memory stalls — the kernel's own word on reclaim pressure.",
		Priority:    30,
		Check:       psiStalls,
	})
}

// psiStalls fires Crit/Warn from kernel-reported PSI avg60 stall ratios —
// the ground truth on whether tasks are actually blocked on reclaim.
func psiStalls(cfg types.Config, r *types.Report, _ []types.Proc) *types.Finding {
	if !r.Pressure.Supported {
		return nil
	}
	switch {
	case r.Pressure.FullAvg60 > cfg.PSIFullCritAvg60:
		return &types.Finding{Severity: types.Crit, Text: fmt.Sprintf(
			"The system is overloaded: every process is blocked waiting for memory (%.0f%% of the last minute).", r.Pressure.FullAvg60)}
	case r.Pressure.SomeAvg60 > cfg.PSISomeWarnAvg60:
		return &types.Finding{Severity: types.Warn, Text: fmt.Sprintf(
			"Processes are periodically waiting for memory (%.0f%% of the last minute), which can cause intermittent slowdowns.", r.Pressure.SomeAvg60)}
	}
	return nil
}

// psiElevated reports whether PSI shows any meaningful stalling, using the
// same thresholds psiStalls itself fires on. Other rules use this to avoid
// contradicting a live PSI warning/critical with an unrelated "this is
// normal" reassurance based on memory accounting alone — cache-not-
// pressure and psi-stalls read different signals, and without this check
// they could both fire at once and tell the user opposite things.
func psiElevated(cfg types.Config, p *types.PSI) bool {
	return p.Supported && (p.FullAvg60 > cfg.PSIFullCritAvg60 || p.SomeAvg60 > cfg.PSISomeWarnAvg60)
}
