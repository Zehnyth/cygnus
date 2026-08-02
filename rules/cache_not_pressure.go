package rules

import (
	"fmt"

	"github.com/zehnyth/cygnus/types"
)

func init() {
	Register(types.Rule{
		Name:        "cache-not-pressure",
		Description: `Flags the classic "RAM looks full but isn't" misread of free/top.`,
		Priority:    20,
		Check:       cacheNotPressure,
	})
}

// cacheNotPressure fires when high reclaimable cache coincides with high
// available memory — reassures against the "used memory looks high" misread.
// Skipped while PSI is showing real stalling: telling the user memory use
// is "normal" at the same time psi-stalls reports the system overloaded
// would be a straight contradiction, not just an unrelated observation.
func cacheNotPressure(cfg types.Config, r *types.Report, _ []types.Proc) *types.Finding {
	if psiElevated(cfg, r.Pressure) {
		return nil
	}
	if r.Mem.ReclaimablePct > cfg.ReclaimableCachePct && r.Mem.AvailablePct > cfg.ReclaimableAvailPct {
		return &types.Finding{Severity: types.Info, Text: fmt.Sprintf(
			"%.0f%% of memory is disk cache, which the kernel reclaims automatically when needed — this is expected, not a shortage.", r.Mem.ReclaimablePct)}
	}
	return nil
}
