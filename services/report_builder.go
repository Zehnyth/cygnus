// Package services contains the business logic — the correlation and
// orchestration layer between raw /proc data (collect), the findings
// engine (rules), and presentation (cli, ui). Neither cli nor ui should
// ever import collect or rules directly; they go through services.
package services

import (
	"sort"
	"strings"

	"github.com/zehnyth/cygnus/collect"
	"github.com/zehnyth/cygnus/rules"
	"github.com/zehnyth/cygnus/types"
)

// Generate runs the full collection + correlation pipeline and returns
// one Report — the single artifact every presentation layer consumes.
func Generate(cfg types.Config, match string, topN int) (*types.Report, error) {
	mem, err := collect.ReadMemInfo()
	if err != nil {
		return nil, err
	}
	procs, err := collect.ListProcs()
	if err != nil {
		return nil, err
	}
	psi := collect.ReadMemPressure()

	return Build(cfg, mem, psi, procs, match, topN), nil
}

// Build applies the registered rules (see the rules package) to
// already-collected data. Split out from Generate so tests can feed in
// fixture data without touching /proc.
func Build(cfg types.Config, mem *types.MemInfo, psi *types.PSI, procs []types.Proc, match string, topN int) *types.Report {
	r := &types.Report{Mem: mem, Pressure: psi, MatchName: match}
	for _, p := range procs {
		if !p.KernelThread {
			r.TotalProcs++
		}
	}

	sort.Slice(procs, func(i, j int) bool { return procs[i].PssKB > procs[j].PssKB })
	// topN <= 0 means "all processes" — used by the TUI, which lists the
	// whole machine and scrolls, rather than a fixed top-N.
	if topN <= 0 || topN > len(procs) {
		topN = len(procs)
	}
	r.TopByPSS = procs[:topN]

	if match != "" {
		for _, p := range procs {
			if strings.Contains(p.Cmdline, match) {
				r.MatchHits++
			}
		}
	}
	for _, p := range procs {
		if p.KernelThread {
			continue // no address space, no PSS expected — not a failure
		}
		if !p.PssOK {
			r.Degraded = true
			r.DegradedCount++
		}
	}

	r.Findings = rules.RunRules(cfg, r, procs)
	if len(r.Findings) == 0 {
		r.Findings = append(r.Findings, types.Finding{Severity: types.OK, Text: "No memory anomalies detected."})
	}
	return r
}
