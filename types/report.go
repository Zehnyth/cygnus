package types

// Report is the single artifact both the CLI text renderer and the TUI
// display. It's also exactly what --json emits — one shape, three views.
type Report struct {
	Mem           *MemInfo  `json:"memory"`
	Pressure      *PSI      `json:"pressure"`
	TopByPSS      []Proc    `json:"top_by_pss"`
	MatchName     string    `json:"match_pattern,omitempty"`
	MatchHits     int       `json:"match_count,omitempty"`
	Findings      []Finding `json:"findings"`
	Degraded      bool      `json:"degraded"`       // true when PSS was unreadable for some processes (not root)
	DegradedCount int       `json:"degraded_count"` // number of processes whose PSS was unreadable
	TotalProcs    int       `json:"total_procs"`    // total real (non-kernel) processes examined
}
