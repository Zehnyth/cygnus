package types

// Rule is one correlation check registered with the rules engine. Its
// Check function reads already-collected data (never /proc directly) and
// returns a Finding, or nil if it doesn't fire.
type Rule struct {
	// Name identifies the rule for Config.DisabledRules and diagnostics.
	// Stable across releases — treat renames as a breaking config change.
	Name string
	// Description is a one-line human explanation, surfaced by future
	// introspection (e.g. a `cygnus rules` listing command).
	Description string
	// Priority controls run/display order, lowest first. Values are spaced
	// by 10 so new rules can be inserted between existing ones without
	// renumbering everything.
	Priority int
	Check    func(cfg Config, r *Report, procs []Proc) *Finding
}
