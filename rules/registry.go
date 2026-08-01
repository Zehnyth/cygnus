// Package rules is the findings engine: a self-registering set of
// correlation checks plus the machinery to run them. Each check lives in
// its own file and registers itself via Register() from init() — adding
// a new check means adding a new file here, never editing this one.
package rules

import (
	"sort"

	"github.com/zehnyth/cygnus/types"
)

// registry collects every rule registered via Register.
var registry []types.Rule

// Register adds a rule to the engine. Called from each rule file's init().
func Register(r types.Rule) {
	registry = append(registry, r)
}

// RunRules executes every enabled rule in priority order and returns the
// findings that fired.
func RunRules(cfg types.Config, r *types.Report, procs []types.Proc) []types.Finding {
	ordered := make([]types.Rule, len(registry))
	copy(ordered, registry)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Priority < ordered[j].Priority })

	disabled := make(map[string]bool, len(cfg.DisabledRules))
	for _, name := range cfg.DisabledRules {
		disabled[name] = true
	}

	var findings []types.Finding
	for _, rule := range ordered {
		if disabled[rule.Name] {
			continue
		}
		if f := rule.Check(cfg, r, procs); f != nil {
			findings = append(findings, *f)
		}
	}
	return findings
}
