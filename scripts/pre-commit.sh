#!/usr/bin/env bash
# Single verification gate for engram commits.
#
# Runs every check that used to be run by hand before a commit: formatting,
# build, vet, race-enabled unit tests, lint, changie fragment validation, and
# a staged-secrets scan.
#
# Usage: ./scripts/pre-commit.sh   (run from the repo root)
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

# Self-install the commit-notification hook (idempotent).
git config core.hooksPath scripts/githooks

start_time=$(date +%s)

section() {
  echo ""
  echo "=== $1 ==="
}

section "gofmt"
fmt_out="$(gofmt -l .)"
if [ -n "$fmt_out" ]; then
  echo "gofmt found unformatted files:"
  echo "$fmt_out"
  exit 1
fi
echo "OK — no unformatted files"

section "go build ./..."
go build ./...

section "go vet ./..."
go vet ./...

section "go test -race ./..."
go test -race ./...

section "golangci-lint"
golangci-lint run ./...

section "changie batch --dry-run patch"
changie_out="$(mktemp)"
trap 'rm -f "$changie_out"' EXIT
if ! changie batch --dry-run patch >"$changie_out" 2>&1; then
  if grep -qi "no unreleased changes" "$changie_out"; then
    echo "NOTICE: no unreleased fragments — nothing to batch"
  else
    cat "$changie_out"
    exit 1
  fi
else
  cat "$changie_out"
fi

section "staged secrets scan"
staged_files="$(git diff --cached --name-only || true)"
bad=""
if [ -n "$staged_files" ]; then
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    case "$f" in
      *.env.example)
        continue
        ;;
    esac
    case "$f" in
      .env | *.env | */.env | *.log | *.dump | data/* | */data/*)
        bad="${bad}${f}"$'\n'
        ;;
    esac
  done <<<"$staged_files"
fi
if [ -n "$bad" ]; then
  echo "BLOCKED: staged files match secret/data patterns:"
  printf '%s' "$bad"
  exit 1
fi
echo "OK — no secrets staged"

end_time=$(date +%s)
echo ""
echo "=== pre-commit checks passed in $((end_time - start_time))s ==="
