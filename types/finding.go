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
}
