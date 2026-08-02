package rules

import (
	"fmt"

	"github.com/zehnyth/cygnus/types"
)

func init() {
	Register(types.Rule{
		Name:        "thread-explosion",
		Description: "Flags a process with a suspiciously high thread count (possible thread leak).",
		Priority:    60,
		Check:       threadExplosion,
	})
}

// threadExplosion fires on the first process whose thread count exceeds
// the configured threshold — a common signature of a thread leak.
func threadExplosion(cfg types.Config, _ *types.Report, procs []types.Proc) *types.Finding {
	for _, p := range procs {
		if p.Threads > cfg.ThreadExplosionCount {
			return &types.Finding{Severity: types.Warn, Text: fmt.Sprintf(
				"%s (PID %d) is running %d threads — unusually high, and often a sign of a thread leak.", p.Comm, p.PID, p.Threads)}
		}
	}
	return nil
}
