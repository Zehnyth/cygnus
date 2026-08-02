package rules

import (
	"fmt"

	"github.com/zehnyth/cygnus/types"
)

func init() {
	Register(types.Rule{
		Name:        "swap-victim",
		Description: "Flags a single process holding a large amount of swapped-out memory.",
		Priority:    40,
		Check:       swapVictim,
	})
}

// swapVictim finds the single process with the most swapped-out memory
// and fires if it exceeds the configured threshold.
func swapVictim(cfg types.Config, _ *types.Report, procs []types.Proc) *types.Finding {
	var worst types.Proc
	for _, p := range procs {
		if p.SwapKB > worst.SwapKB {
			worst = p
		}
	}
	if worst.SwapKB > cfg.SwapVictimKB {
		return &types.Finding{Severity: types.Warn, Text: fmt.Sprintf(
			"%s (PID %d) has %.1f GB swapped to disk, which will slow it down. This typically happens after a period of low memory.", worst.Comm, worst.PID, float64(worst.SwapKB)/1048576)}
	}
	return nil
}
