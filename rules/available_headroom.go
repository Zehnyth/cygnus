package rules

import (
	"fmt"

	"github.com/zehnyth/cygnus/types"
)

func init() {
	Register(types.Rule{
		Name:        "available-headroom",
		Description: "Flags low or critical available-memory headroom.",
		Priority:    10,
		Check:       availableHeadroom,
	})
}

// availableHeadroom fires Crit/Warn as AvailablePct drops below the
// configured thresholds — the earliest signal of OOM-kill risk.
func availableHeadroom(cfg types.Config, r *types.Report, _ []types.Proc) *types.Finding {
	switch {
	case r.Mem.AvailablePct < cfg.AvailableCritPct:
		return &types.Finding{Severity: types.Crit, Text: fmt.Sprintf(
			"Memory is nearly exhausted (%.1f%% free). The kernel may start killing processes to reclaim memory.", r.Mem.AvailablePct)}
	case r.Mem.AvailablePct < cfg.AvailableWarnPct:
		return &types.Finding{Severity: types.Warn, Text: fmt.Sprintf(
			"Available memory is low (%.1f%% free).", r.Mem.AvailablePct)}
	}
	return nil
}
