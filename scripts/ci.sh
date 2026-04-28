#!/usr/bin/env bash
#
# scripts/ci.sh — single local entry point for digital.vasic.mdns CI.
#
# Per CONSTITUTION.md, this module does NOT use hosted CI services.
#
# Usage:
#   scripts/ci.sh              # full gate
#   scripts/ci.sh --quick      # skip gosec / govulncheck

set -Eeuo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

QUICK=false
for arg in "$@"; do
  case "$arg" in
    --quick) QUICK=true ;;
    -h|--help) sed -n '/^# Usage/,/^# *$/p' "$0" | sed 's/^# \?//'; exit 0 ;;
    *) echo "unknown flag: $arg" >&2; exit 2 ;;
  esac
done

log()  { printf '\033[1;36m[ci]\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31m[ci:fail]\033[0m %s\n' "$*" >&2; exit 1; }

# 1. Tidy invariant
log "step 1/6  go mod tidy invariant"
_pre="$(sha256sum go.mod go.sum 2>/dev/null | sort)"
go mod tidy
_post="$(sha256sum go.mod go.sum 2>/dev/null | sort)"
[[ "$_pre" == "$_post" ]] || fail "go mod tidy produced a diff; commit the tidied result"

# 2. Vet
log "step 2/6  go vet ./..."
go vet ./...

# 3. Build
log "step 3/6  go build ./..."
go build ./...

# 4. Test
log "step 4/6  go test -race -count=1 ./..."
go test -race -count=1 ./...

if $QUICK; then
  log "ci OK (quick)"
  exit 0
fi

# 5. gosec
if command -v gosec >/dev/null 2>&1; then
  log "step 5/6  gosec ./..."
  gosec -quiet ./...
else
  log "step 5/6  gosec not installed; skipping"
fi

# 6. govulncheck
if command -v govulncheck >/dev/null 2>&1; then
  log "step 6/6  govulncheck ./..."
  govulncheck ./...
else
  log "step 6/6  govulncheck not installed; skipping"
fi

log "ci OK"
