// Package types holds every domain data shape shared across the codebase
// (collect, rules, services, cli, ui) — pure structs and enums only, no
// logic, no I/O, no dependency on any other internal package. If you're
// defining a `type X struct` anywhere else in the repo, it likely belongs
// here instead.
package types

// Config holds every tunable threshold the findings engine uses, plus
// which rules are disabled. It is the "injected preferences" surface —
// services.DefaultConfig() gives the built-in behavior, and
// services.LoadConfig() lets an operator override individual fields via
// a JSON file (e.g. one dropped at /etc/cygnus/config.json by a package
// install) without touching code.
type Config struct {
	// AvailableCritPct/AvailableWarnPct: available-headroom rule thresholds,
	// percent of total memory still available.
	AvailableCritPct float64 `json:"available_crit_pct"`
	AvailableWarnPct float64 `json:"available_warn_pct"`

	// ReclaimableCachePct/ReclaimableAvailPct: cache-not-pressure rule
	// thresholds — fires when reclaimable cache is high AND available
	// memory is still high.
	ReclaimableCachePct float64 `json:"reclaimable_cache_pct"`
	ReclaimableAvailPct float64 `json:"reclaimable_avail_pct"`

	// PSIFullCritAvg60/PSISomeWarnAvg60: psi-stalls rule thresholds, read
	// from /proc/pressure/memory avg60 fields.
	PSIFullCritAvg60 float64 `json:"psi_full_crit_avg60"`
	PSISomeWarnAvg60 float64 `json:"psi_some_warn_avg60"`

	// SwapVictimKB: swap-victim rule fires when a single process has at
	// least this much swapped out, in KiB.
	SwapVictimKB uint64 `json:"swap_victim_kb"`

	// RSSPSSGapRatio: rss-pss-gap rule fires when summed RSS overstates
	// summed PSS by more than this ratio (1.5 = 50% overstatement).
	RSSPSSGapRatio float64 `json:"rss_pss_gap_ratio"`

	// ThreadExplosionCount: thread-explosion rule fires when any process
	// has at least this many threads.
	ThreadExplosionCount int `json:"thread_explosion_count"`

	// DisabledRules lists rule Names (see Rule.Name) to skip entirely.
	DisabledRules []string `json:"disabled_rules,omitempty"`
}
