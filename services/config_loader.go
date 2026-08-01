package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/zehnyth/cygnus/types"
)

// DefaultConfig returns cygnus's built-in thresholds — the values every
// rule uses unless overridden by LoadConfig.
func DefaultConfig() types.Config {
	return types.Config{
		AvailableCritPct:     5,
		AvailableWarnPct:     15,
		ReclaimableCachePct:  30,
		ReclaimableAvailPct:  25,
		PSIFullCritAvg60:     1,
		PSISomeWarnAvg60:     5,
		SwapVictimKB:         512 * 1024,
		RSSPSSGapRatio:       1.5,
		ThreadExplosionCount: 2000,
	}
}

// LoadConfig starts from DefaultConfig() and overlays JSON fields found at
// path. A missing file is not an error — cygnus runs on defaults, which
// is the expected case for most installs. An empty path always returns
// defaults without touching the filesystem.
func LoadConfig(path string) (types.Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}
	return cfg, nil
}
