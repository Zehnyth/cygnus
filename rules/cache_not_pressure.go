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
func cacheNotPressure(cfg types.Config, r *types.Report, _ []types.Proc) *types.Finding {
	if r.Mem.ReclaimablePct > cfg.ReclaimableCachePct && r.Mem.AvailablePct > cfg.ReclaimableAvailPct {
		return &types.Finding{Severity: types.Info, Text: fmt.Sprintf(
			"%.0f%% of RAM is reclaimable cache/buffers — 'used' memory here is not pressure.", r.Mem.ReclaimablePct)}
	}
	return nil
}
