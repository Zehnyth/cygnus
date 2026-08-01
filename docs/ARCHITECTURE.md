# Architecture

How a `cygnus report` invocation moves through the codebase, end to end.

## Entry point

**[`main.go`](../main.go)** is the only entry point. It does one thing:
call `cli.Execute()` and exit non-zero on error. All real work happens
inside the `cli` package and below.

## Package graph

Dependencies flow one direction only — enforced by import discipline, not
tooling. `types` is the leaf: it imports nothing internal and nothing
external beyond the stdlib.

```mermaid
graph TD
    main["main.go<br/>(entry point)"] --> cli
    cli["cli/<br/>Cobra commands, flags, text & JSON output"] --> services
    ui["ui/<br/>Bubble Tea TUI"] --> types
    cli -.->|"tea.NewProgram(ui.New(report))"| ui
    services["services/<br/>orchestration: Generate, Build, Config loading"] --> collect
    services --> rules
    collect["collect/<br/>reads /proc — the only package that does"] --> types
    rules["rules/<br/>self-registering findings engine"] --> types
    types["types/<br/>every domain shape — the leaf"]

    style types fill:#0b3d2e,stroke:#10B981,color:#F5EFE0
    style collect fill:#123a2e,stroke:#10B981,color:#F5EFE0
    style rules fill:#123a2e,stroke:#10B981,color:#F5EFE0
    style services fill:#1a2f4a,stroke:#5B9BD5,color:#F5EFE0
    style cli fill:#3a2a12,stroke:#D4AF37,color:#F5EFE0
    style ui fill:#3a2a12,stroke:#D4AF37,color:#F5EFE0
    style main fill:#4a1a1a,stroke:#E06C82,color:#F5EFE0
```

`cli` and `ui` never import `collect` or `rules` directly — everything
goes through `services`. That's not a suggestion; nothing in `cli`/`ui`
has a reason to import them, so the boundary holds naturally.

## Request flow: `cygnus report`

```mermaid
sequenceDiagram
    participant User
    participant cli as cli (root.go / report_command.go)
    participant svc as services (report_builder.go)
    participant col as collect
    participant proc as /proc
    participant rul as rules (registry.go)
    participant out as text_renderer.go / ui / json

    User->>cli: sudo cygnus report [--tui|--json] [--top N] [--config path]
    cli->>cli: resolveConfigPath() — flag > XDG > ~/.config > /etc
    cli->>svc: LoadConfig(path)
    svc-->>cli: types.Config (defaults + overrides)
    cli->>svc: Generate(cfg, match, topN)
    svc->>col: ReadMemInfo()
    col->>proc: read /proc/meminfo
    proc-->>col: raw text
    col-->>svc: types.MemInfo
    svc->>col: ListProcs()
    col->>proc: walk /proc/<pid>/{comm,cmdline,status,statm,smaps_rollup,fd}
    proc-->>col: raw text (per pid)
    col-->>svc: []types.Proc
    svc->>col: ReadMemPressure()
    col->>proc: read /proc/pressure/memory
    proc-->>col: raw text
    col-->>svc: types.PSI
    svc->>svc: Build() — sort by PSS, truncate to topN,<br/>count degraded/kernel-thread procs
    svc->>rul: RunRules(cfg, report, procs)
    rul->>rul: sort registry by Priority,<br/>skip Config.DisabledRules,<br/>call each Rule.Check
    rul-->>svc: []types.Finding
    svc-->>cli: *types.Report
    cli->>out: renderText(w, report) OR ui.New(report) OR json.Encode(report)
    out-->>User: text / interactive TUI / JSON
```

## Package responsibilities

| Package | Owns | Never does |
| --- | --- | --- |
| [`types/`](../types) | Every domain shape: `MemInfo`, `PSI`, `Proc`, `Finding`, `Report`, `Config`, `Rule`. | Logic, I/O, imports of any other internal package. |
| [`collect/`](../collect) | Reading and parsing `/proc`. One `*_reader.go` per source, each splitting I/O (`ReadX`) from pure parsing (`parseX`/`readX`). | Correlation, thresholds, rendering. |
| [`rules/`](../rules) | The findings engine: `registry.go` (register/run) + one self-registering file per check. | Collecting data, rendering, orchestration. |
| [`services/`](../services) | Orchestration: wiring `collect` + `rules` into a `Report` (`report_builder.go`); config loading behavior (`config_loader.go`). | Rule logic, type definitions, `/proc` access. |
| [`cli/`](../cli) | Cobra command/flag wiring, text rendering, config-path resolution. | Touching `/proc` or `rules` directly. |
| [`ui/`](../ui) | Bubble Tea rendering of a `*types.Report`. | Touching `/proc`, correlation, mutating the report. |
| [`utils/`](../utils) | Stateless unit-conversion helpers (KiB → GB/MB). | Everything else — imports nothing internal. |

## Extending Cygnus

**Add a new finding:** create `rules/your_check.go` with a `Check` function
and an `init()` that calls `Register(types.Rule{...})`. No other file
changes — see [`rules/thread_explosion.go`](../rules/thread_explosion.go)
for the shortest example.

**Add a new tunable threshold:** add a field to `types.Config`
([`types/config.go`](../types/config.go)), a default in
`services.DefaultConfig()` ([`services/config_loader.go`](../services/config_loader.go)),
and reference `cfg.YourField` from the rule that needs it.

**Add a new `/proc` source:** add `collect/your_reader.go` following the
`ReadX` (I/O) / `parseX` (pure) split used by the existing readers, and a
matching shape in `types/`.

**Add a new TUI panel:** add a `renderYourPanel` function in
[`ui/view.go`](../ui/view.go) and call it from `Model.layout()` in
[`ui/model.go`](../ui/model.go); pull colors from the `Tier`/`tierPalette`
table in [`ui/theme.go`](../ui/theme.go) rather than hardcoding new ones.
