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
			"Sustained full memory stalls (PSI full avg60=%.2f) — the whole system is waiting on reclaim.", r.Pressure.FullAvg60)}
	case r.Pressure.SomeAvg60 > cfg.PSISomeWarnAvg60:
		return &types.Finding{Severity: types.Warn, Text: fmt.Sprintf(
			"Tasks are stalling on memory (PSI some avg60=%.2f).", r.Pressure.SomeAvg60)}
	}
	return nil
}
