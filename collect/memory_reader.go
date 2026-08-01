// Package collect is the only layer that reads /proc. Each file here
// pairs a ReadX function (real file I/O) with a pure parseX/readX helper
// (fixture-testable, no I/O) — see CLAUDE.md for the full convention.
package collect

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/zehnyth/cygnus/types"
)

// ReadMemInfo reads and parses /proc/meminfo.
func ReadMemInfo() (*types.MemInfo, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return parseMemInfo(f)
}

// parseMemInfo is the pure parsing half of ReadMemInfo, split out so it
// can be exercised with fixture content instead of a real /proc/meminfo.
func parseMemInfo(r io.Reader) (*types.MemInfo, error) {
	m := &types.MemInfo{}
	fields := map[string]*uint64{
		"MemTotal":     &m.TotalKB,
		"MemFree":      &m.FreeKB,
		"MemAvailable": &m.AvailableKB,
		"Buffers":      &m.BuffersKB,
		"Cached":       &m.CachedKB,
		"SReclaimable": &m.SReclaimKB,
		"SUnreclaim":   &m.SUnreclaimKB,
		"Shmem":        &m.ShmemKB,
		"SwapTotal":    &m.SwapTotalKB,
		"SwapFree":     &m.SwapFreeKB,
		"Dirty":        &m.DirtyKB,
	}

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		key, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		dst, want := fields[key]
		if !want {
			continue
		}
		numStr := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(rest), "kB"))
		if v, err := strconv.ParseUint(strings.TrimSpace(numStr), 10, 64); err == nil {
			*dst = v
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	if m.TotalKB > 0 {
		m.AvailablePct = 100 * float64(m.AvailableKB) / float64(m.TotalKB)
		m.ReclaimablePct = 100 * float64(m.BuffersKB+m.CachedKB+m.SReclaimKB) / float64(m.TotalKB)
	}
	return m, nil
}
