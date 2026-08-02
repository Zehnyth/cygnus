package types

// Severity is the plain-language urgency level of a Finding.
type Severity string

const (
	OK   Severity = "ok"
	Warn Severity = "warning"
	Crit Severity = "critical"
	Info Severity = "info"
)

// Finding is one plain-language observation produced by a rule.
type Finding struct {
	Severity Severity `json:"severity"`
	Text     string   `json:"text"`
	// Rule is the originating Rule.Name (see rules.Register), letting
	// callers trace a finding back to what produced it — e.g. so the TUI
	// can key a panel's status off the same rule-engine truth as the
	// FINDINGS list, instead of recomputing severity independently with a
	// second set of thresholds that can disagree with the first.
	Rule string `json:"rule,omitempty"`
}
