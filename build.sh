#!/usr/bin/env bash
# Builds cygnus into bin/cygnus. Mirrors the manual steps in README.md.
#
# Usage:
#   ./build.sh              # go mod tidy + build -> bin/cygnus
#   ./build.sh --install    # also install to /usr/local/bin (uses sudo)
#   ./build.sh --clean      # remove bin/ before building
#   ./build.sh --no-color   # plain output (also respects NO_COLOR env var)

set -euo pipefail

BIN_DIR="bin"
BIN_NAME="cygnus"
INSTALL=false
CLEAN=false
NO_COLOR_FLAG=false

for arg in "$@"; do
  case "$arg" in
    --install)  INSTALL=true ;;
    --clean)    CLEAN=true ;;
    --no-color) NO_COLOR_FLAG=true ;;
    -h|--help)
      echo "Usage: $0 [--install] [--clean] [--no-color]"
      exit 0
      ;;
    *)
      echo "Unknown option: $arg" >&2
      exit 1
      ;;
  esac
done

# ---- Presentation ----------------------------------------------------
# Color and the spinner only activate on a real terminal, and never when
# NO_COLOR is set or --no-color is passed — keeps CI/log output plain.
if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "$NO_COLOR_FLAG" = false ]; then
  BOLD=$'\033[1m'; DIM=$'\033[2m'
  BLUE=$'\033[38;5;75m'; CYAN=$'\033[38;5;123m'
  GREEN=$'\033[38;5;114m'; RED=$'\033[38;5;203m'
  RESET=$'\033[0m'
  ANIMATE=true
else
  BOLD=""; DIM=""; BLUE=""; CYAN=""; GREEN=""; RED=""
  RESET=""
  ANIMATE=false
fi

banner() {
  printf "%s%s✦ CYGNUS ✦%s  %shost memory triage — build%s\n" \
    "$BLUE" "$BOLD" "$RESET" "$DIM" "$RESET"
  printf "%s────────────────────────────────────────%s\n\n" "$BLUE" "$RESET"
}

ok()   { printf "  %s✔%s %s %s(%ss)%s\n" "$GREEN" "$RESET" "$1" "$DIM" "$2" "$RESET"; }
fail() { printf "  %s✖%s %s\n" "$RED" "$RESET" "$1"; }

# Runs a command. On a real terminal, shows a spinner and hides output
# unless the command fails; otherwise prints a plain "> desc" line.
run() {
  local desc="$1"; shift
  local log start elapsed pid status
  log="$(mktemp)"
  start=$(date +%s)

  "$@" >"$log" 2>&1 &
  pid=$!

  if [ "$ANIMATE" = true ]; then
    local frames='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local i=0 idx
    while kill -0 "$pid" 2>/dev/null; do
      idx=$(( i % ${#frames} ))
      printf "\r%s%s%s %s" "$CYAN" "${frames:idx:1}" "$RESET" "$desc"
      i=$((i + 1))
      sleep 0.08
    done
    printf "\r\033[K"
  else
    printf "> %s\n" "$desc"
  fi

  status=0
  wait "$pid" || status=$?
  elapsed=$(( $(date +%s) - start ))

  if [ "$status" -eq 0 ]; then
    ok "$desc" "$elapsed"
  else
    fail "$desc"
    printf "  %s---- output ----%s\n" "$DIM" "$RESET"
    cat "$log"
    rm -f "$log"
    exit "$status"
  fi
  rm -f "$log"
}

banner

if ! command -v go >/dev/null 2>&1; then
  fail "go not found on PATH"
  printf "  Install it from %shttps://go.dev/dl/%s (see README.md).\n" "$CYAN" "$RESET"
  exit 1
fi
printf "  %s%s%s\n\n" "$DIM" "$(go version)" "$RESET"

[ "$CLEAN" = true ] && run "Cleaning $BIN_DIR/" rm -rf "$BIN_DIR"

run "Tidying dependencies (go mod tidy)" go mod tidy

mkdir -p "$BIN_DIR"
run "Compiling $BIN_DIR/$BIN_NAME" go build -o "$BIN_DIR/$BIN_NAME" .

SIZE=$(du -h "$BIN_DIR/$BIN_NAME" | cut -f1)

if [ "$INSTALL" = true ]; then
  run "Installing to /usr/local/bin/$BIN_NAME (sudo)" \
    sudo install -m 755 "$BIN_DIR/$BIN_NAME" "/usr/local/bin/$BIN_NAME"
fi

echo
printf "%s%s✔ Build complete%s  %s%s/%s (%s)%s\n" \
  "$GREEN" "$BOLD" "$RESET" "$DIM" "$BIN_DIR" "$BIN_NAME" "$SIZE" "$RESET"
echo
if [ "$INSTALL" = true ]; then
  printf "  Installed to /usr/local/bin/%s\n" "$BIN_NAME"
  printf "  Run:      %ssudo %s report%s\n" "$CYAN" "$BIN_NAME" "$RESET"
else
  printf "  Run:      %ssudo ./%s/%s report%s\n" "$CYAN" "$BIN_DIR" "$BIN_NAME" "$RESET"
  printf "  Install:  %s./build.sh --install%s\n" "$CYAN" "$RESET"
fi
printf "  %s(sudo is the recommended default — full PSS/FD data needs root; see README)%s\n" "$DIM" "$RESET"