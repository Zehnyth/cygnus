# Cygnus

**Host memory triage for Linux, in one command.**

Cygnus reads `/proc` directly and correlates memory, PSI pressure, and
per-process data into a single report with plain-language findings —
instead of the usual `free` + `ps --sort=-rss` + `lsof` + `slabtop`
pipeline, run separately, read separately, and correlated by hand.

```
MEMORY  total 15.6 GB · available 2.1 GB (13.5%) · cache+buffers 3.2 GB · slab 0.9 GB · dirty 12 MB
PSI     some avg10=0.42 avg60=1.10 · full avg10=0.05 avg60=0.20

FINDINGS
  [warning] 13.5% memory available — headroom is low.
  [warning] chrome (pid 4821) has 2143 threads — possible thread leak.

TOP BY PSS
  PID    USER  PSS     RSS     SWAP    THR   FDS  COMMAND
  4821   fran  1.8 GB  2.4 GB  0 MB    2143  312  chrome
  9102   fran  640 MB  980 MB  120 MB  24    88   node
```

## Why

Cygnus doesn't invent new primitives — it correlates two that already
exist and are already proven, and adds the layer neither one provides
on its own.

**PSS, not RSS.** This is the same insight
[`smem`](https://www.selenic.com/smem/) has shipped for over a decade:
summing RSS across processes double-counts shared pages — libraries,
mmap'd files, forked children sharing the same segments — and overstates
real usage. PSS attributes shared memory fairly across the processes
actually using it. If you already trust `smem` for this, Cygnus uses the
same idea, not a new one.

**PSI, not load average.** [PSI (Pressure Stall
Information)](https://docs.kernel.org/accounting/psi.html) is the same
kernel signal [`systemd-oomd`](https://man7.org/linux/man-pages/man8/systemd-oomd.service.8.html)
already uses to kill processes before the kernel OOM killer wakes up,
and the same one Android's low-memory killer uses to decide what to
evict. On most modern systemd-based distros it's already running in the
background. Cygnus reads it; it doesn't reimplement it.

**What neither of those tools does** is what Cygnus is actually for.
`smem` is a data reporter — no verdicts, no thresholds, no PSI. `systemd-oomd`
is a silent background daemon — it kills, it doesn't report, and you
can't point it at a terminal or a CI pipeline. Cygnus sits between them:
one command that reads both signals, correlates them against
per-process data, and turns raw numbers into plain-language findings —
with a rules engine you can tune, and an output contract (text, TUI, or
JSON) built for a human at a terminal and an `assert` task in CI alike.

## Features

- **One report, three views** — the same data as human-readable text, an
  interactive terminal UI (`--tui`), or JSON (`--json`) for scripts, CI
  gates, and Ansible `assert` tasks.
- **Plain-language findings**, not just numbers — a small rules engine
  (see [Configuration](#configuration)) turns raw metrics into statements
  like *"chrome (pid 4821) has 2143 threads — possible thread leak."*
- **Degrades gracefully without root** — runs unprivileged and tells you
  exactly what it couldn't read, rather than failing or guessing.
- **Configurable thresholds** via an optional JSON file — tune what counts
  as "low headroom" or a "thread explosion" without recompiling.
- **Zero shell-outs** — reads `/proc` directly; never invokes `free`,
  `ps`, `lsof`, or any other external tool.

## Requirements

- **Linux** — reads `/proc/meminfo`, `/proc/pressure/memory`, and
  per-process `/proc/<pid>/...` files directly. No other OS is supported.
- **Go**, to build from source (see below). Go has no formal LTS track —
  each major version is supported until two newer majors ship, so build
  with whatever the **current stable release** is. This project targets
  `go 1.23` as a floor (see `go.mod`).

## Installation

### 1. Install Go

1. Download the current stable release for your platform from
   [go.dev/dl](https://go.dev/dl/).
2. Follow the [official install instructions](https://go.dev/doc/install)
   for your OS.
3. Confirm it's on your `PATH`:

   ```bash
   go version
   # go version go1.26.x linux/amd64
   ```

   On Fedora, `sudo dnf install golang` also works, but the repo version
   can lag upstream — prefer the official tarball if you want the latest
   release.

### 2. Build

```bash
git clone https://github.com/zehnyth/cygnus.git
cd cygnus
./build.sh
```

This runs `go mod tidy` then builds to `bin/cygnus` (`bin/` is gitignored
— it's build output, not source). Options:

```bash
./build.sh --install   # also install to /usr/local/bin (uses sudo)
./build.sh --clean     # remove bin/ before building
./build.sh --no-color  # plain output, no spinner (also respects NO_COLOR)
```

Or by hand — this is exactly what `build.sh` runs:

```bash
go mod tidy
go build -o bin/cygnus .
sudo install -m 755 bin/cygnus /usr/local/bin/cygnus   # optional
```

## Usage

> [!IMPORTANT]
> **Run cygnus with `sudo` by default.** Full per-process PSS and open
> file descriptor counts come from `/proc/<pid>/smaps_rollup` and
> `/proc/<pid>/fd`, both root-only for processes you don't own. Without
> root, cygnus still runs and produces a valid report — it falls back to
> RSS, marks FDs as `?`, and flags exactly how many processes were
> affected — but on a normal desktop or server most processes aren't
> yours, so the report will be mostly degraded. `sudo` is the default,
> not an edge case.

```bash
# human-readable report (recommended default)
sudo cygnus report

# interactive scrollable view — auto-refreshes on a timer;
# press p to pause/resume, q to quit
sudo cygnus report --tui

# JSON, for scripts / Ansible / CI
sudo cygnus report --json

# also count processes matching a substring in their cmdline
sudo cygnus report --match nginx

# change how many top-PSS processes are listed (default 20)
sudo cygnus report --top 10
```

If you built without `--install`, run `sudo ./bin/cygnus report` instead
(or `sudo ./build.sh` — see [Installation](#installation)).

### Flags

| Flag       | Applies to | Description                                                             |
| ---------- | ---------- | ------------------------------------------------------------------------ |
| `--json`   | root       | Emit the report as machine-readable JSON instead of text.               |
| `--config` | root       | Path to a JSON config file overriding rule thresholds. See below.       |
| `--tui`    | `report`   | Launch the interactive scrollable view instead of static text.          |
| `--top N`  | `report`   | Number of top-PSS processes to list (default `20`; ignored under `--tui` unless explicitly set, which then lists every process). |
| `--match S`| `report`   | Also count processes whose cmdline contains substring `S`.              |

## Configuration

Cygnus's findings are produced by a small rules engine with tunable
thresholds — no recompiling needed to adjust what counts as "low
headroom" or a "thread explosion." Provide overrides as JSON:

```json
{
  "available_warn_pct": 20,
  "thread_explosion_count": 500,
  "disabled_rules": ["cache-not-pressure"]
}
```

Only the fields you set are overridden; everything else keeps its
built-in default. Cygnus looks for a config file in this order:

1. `--config /path/to/config.json` (explicit)
2. `$XDG_CONFIG_HOME/cygnus/config.json`
3. `~/.config/cygnus/config.json`
4. `/etc/cygnus/config.json` (system-wide — where a package install would
   drop one)
5. none found → built-in defaults

See [`types/config.go`](types/config.go) for every available field and
[`rules/`](rules/) for what each threshold controls.

## Architecture

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for a full flow diagram
— entry point, package responsibilities, and how a `cygnus report`
invocation moves through the codebase. Short version: `main` → `cli` →
`services` → (`collect` + `rules`) → `types`, one direction only.

## Contributing

Branches follow `<TICKET>--<kebab-description>`. Run `golangci-lint run`
before sending changes (config: `.golangci.yml`). See
[`CLAUDE.md`](CLAUDE.md) for the full architectural conventions this
codebase enforces.