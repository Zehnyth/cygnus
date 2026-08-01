# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What Cygnus is

A Linux host memory-triage CLI. It reads `/proc` directly (never shells out to `free`/`ps`/`lsof`) and correlates memory, PSI pressure, and per-process data into one report with plain-language findings. Central thesis: it reports **PSS** (proportional set size), not RSS, because summing RSS double-counts shared pages. Linux-only — `/proc` is read unconditionally, with no build tag guarding it.

## Build & run

- Build with `./build.sh` (runs `go mod tidy` then `go build -o bin/cygnus .`). Flags: `--install` (`sudo install` to `/usr/local/bin`), `--clean`, `--no-color`.
- Lint with `golangci-lint run` (config: `.golangci.yml`).
- Full data requires **root**: PSS comes from `/proc/<pid>/smaps_rollup` and FD counts from `/proc/<pid>/fd`. Run the binary with `sudo` when verifying real output.
- Without root it degrades gracefully — don't treat missing PSS/FD data as a bug: `Proc.PssOK=false`, `PssKB` falls back to `RssKB`, `FDs=-1` (shown as `?`), and `Report.Degraded`/`DegradedCount` are set.

## Architecture — enforced layering

Dependency flow is strict and enforced by imports: `main.go` → `cli` → `services` → `collect` (and `rules`) → `types` (the leaf, imports nothing internal). `utils` is stateless helpers importing nothing.

Six packages, each holding exactly one kind of thing:

- **`types/`** — every domain data shape used across the codebase, and nothing else. No logic, no I/O. `memory_info.go` (`MemInfo`), `memory_pressure.go` (`PSI`), `process.go` (`Proc`, `ProcStatus`), `finding.go` (`Severity`, `Finding`), `report.go` (`Report`), `config.go` (`Config`), `rule.go` (`Rule`). If you're about to define a `type X struct` anywhere else in the repo, it almost certainly belongs here instead — the one exception is presentation-only types with zero callers outside `ui` (see the `Tier` note below).
- **`collect/`** — the ONLY layer that reads `/proc`. One reader file per source (`memory_reader.go`, `pressure_reader.go`, `process_reader.go`), each splitting **I/O from parsing**: `ReadX()` opens the real file, a lowercase `parseX(io.Reader)` (or, in `process_reader.go`, per-field `readStatus`/`readStatm`/`readSmapsRollup`/`readFDCount` helpers taking a `base` directory) does the actual parsing. The parse side is pure/fixture-testable; only the `ReadX` wrappers touch a real path.
- **`rules/`** — the findings engine: a self-registering set of correlation checks plus the registry that runs them (see below). Only rule logic lives here.
- **`services/`** — pure orchestration: `report_builder.go` (`Generate`/`Build`, wiring `collect` + `rules` + `types` together) and `config_loader.go` (`DefaultConfig`/`LoadConfig` behavior — the `Config` shape itself lives in `types`). No rule logic, no type definitions.
- **`cli/`** — Cobra command/flag wiring (`root.go`, `report_command.go`), the text renderer (`text_renderer.go`), and config-path resolution (`config_resolver.go`). **`cli` and `ui` must NEVER import `collect` or `rules` directly** — they go through `services`.
- **`ui/`** — renders a `*types.Report` with Bubble Tea (`model.go`, `view.go`, `theme.go`); never reads `/proc`.

### The findings engine is a self-registering rule registry, not a fixed list

Each check is a `types.Rule{Name, Description, Priority, Check}` living in its own `rules/*.go` file (e.g. `rules/swap_victim.go`), registered via `rules.Register(...)` inside that file's `init()`. **Adding a new check means adding a new file to `rules/` — no other file changes.** `Priority` (spaced by 10: 10, 20, 30...) controls run/display order; leave gaps so a new rule can be inserted between two existing ones without renumbering. `rules.RunRules` (in `rules/registry.go`) sorts by `Priority`, skips any `Rule.Name` present in `Config.DisabledRules`, and returns the fired findings — this is what `services.Build` calls.

`Rule.Check` takes `(types.Config, *types.Report, []types.Proc)` — thresholds come from `Config`, not literals. `services.Generate`/`Build` both take a `types.Config` as their first argument.

### Config: how thresholds get customized

`services.DefaultConfig()` is the built-in behavior. `services.LoadConfig(path)` overlays JSON onto the defaults (into a `types.Config`) — a missing file is not an error (defaults apply). `cli/config_resolver.go`'s `resolveConfigPath` picks the file: `--config` flag > `$XDG_CONFIG_HOME/cygnus/config.json` > `~/.config/cygnus/config.json` > `/etc/cygnus/config.json` > none (defaults). The `/etc` path is there for a future RPM install to drop a system-wide default without the user having to do anything. This is the extension point for "customize behavior without recompiling" — deliberately *not* Go's `plugin` package or dynamically loaded `.so` files, which require exact toolchain/dependency version matches between host and plugin and are considered fragile in the Go ecosystem; a single static binary is what you want for an RPM anyway.

## Gotchas

- **Kernel threads:** a process with no `VmRSS` line in `/proc/<pid>/status` is a kernel thread (`KernelThread=true`). They legitimately have no PSS, so exclude them from the degraded count and `TotalProcs` (counting them inflated the degraded count — a fixed bug).
- **Single `Report`, three views:** the one `types.Report` shape is rendered as text (`cli.renderText` in `cli/text_renderer.go`, takes an `io.Writer`), TUI (`ui`), and `--json` (verbatim encoder). Keep all `json` tags intact when editing `types`.
- **Units:** everything internal is KiB (`uint64`). `utils.GB` divides by 1048576, `utils.MB` by 1024.
- **TUI colors go through the `Tier` enum, not ad hoc switches:** `ui/theme.go` defines `type Tier int` (`TierGood`/`TierWarn`/`TierCrit`) with a single `tierPalette map[Tier]lipgloss.AdaptiveColor` — that's the one place "what color means what" is decided. `severityTier` maps `types.Severity` → `Tier` for verdict/badge coloring; `severityStyles` gives each `Severity` its exact badge bg/fg (Info is muted, not "good"-green, so it isn't collapsed into `Tier`). Threshold-based panels (`usageTier`, `psiTier` in `ui/view.go`) each bucket their own metric into a `Tier` and then pull the color from `tierPalette` — add new threshold-colored UI by writing a small `xTier(...) Tier` function, not a new color switch. `Tier`/`severityStyle` deliberately stay in `ui/theme.go` rather than `types/` — they're presentation-only, have zero callers outside `ui`, and are tied to the external `lipgloss` library, which the shape-only `types` package must not depend on. Every base palette color is still a `lipgloss.AdaptiveColor` with distinct Light/Dark values (never assume the background).
- **`--top` + `--tui`:** when `--tui` is set and `--top` wasn't explicitly passed, `effectiveTop` is forced to `0` (all processes); `topN <= 0` means "all" in `services.Build`.

## Tests

None exist yet — removed pending a design decision on test layout (Go requires `_test.go` files testing unexported code to live in the same directory as that code, which constrains how far tests can be split from source). Add real tests when that design lands; until then don't assume any behavior is covered.

## Git

Branches follow `<TICKET>--<kebab-description>` (e.g. `CYG-1--bootstrap-cygnus-cli-project`). **Do not commit or push** — the user reviews all changes and makes commits/pushes themselves.
