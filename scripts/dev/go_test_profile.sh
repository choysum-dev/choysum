#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
#
# Profile `go test` wall-clock, per-package elapsed, and slow tests.
# Default command matches local developer feel: no -count=1, no -cover.
# Compare CI numbers separately (those jobs use -count=1 -covermode=atomic).
#
# Usage:
#   ./scripts/dev/go_test_profile.sh
#   ./scripts/dev/go_test_profile.sh ./pkg/oerrors
#   SCOPE='./cmd ./pkg/oerrors' ./scripts/dev/go_test_profile.sh
#   ./scripts/dev/go_test_profile.sh --summarize /tmp/choysum-go-test-profile/go-test.jsonl
#
# Env:
#   SCOPE            default: ./...  (ignored when packages are passed as args)
#   PROFILE_DIR      default: /tmp/choysum-go-test-profile
#   GO_TEST_FLAGS    extra `go test` flags (script always adds -json)
#   SLOW_SECS        default: 0.5
#   PKG_TOP          default: 20
#   TEST_TOP         default: 40

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

SUMMARIZE="$ROOT/scripts/dev/go_test_profile_summarize.py"
PROFILE_DIR="${PROFILE_DIR:-/tmp/choysum-go-test-profile}"
SLOW_SECS="${SLOW_SECS:-0.5}"
PKG_TOP="${PKG_TOP:-20}"
TEST_TOP="${TEST_TOP:-40}"

summarize_file() {
  local jsonl="$1"
  local timing="${2:-}"
  if [[ -n "$timing" ]]; then
    python3 "$SUMMARIZE" "$jsonl" --slow "$SLOW_SECS" --pkg-top "$PKG_TOP" --test-top "$TEST_TOP" --timing "$timing"
  else
    python3 "$SUMMARIZE" "$jsonl" --slow "$SLOW_SECS" --pkg-top "$PKG_TOP" --test-top "$TEST_TOP"
  fi
}

if [[ "${1:-}" == "--summarize" ]]; then
  if [[ $# -lt 2 ]]; then
    echo "usage: $0 --summarize <go-test.jsonl> [timing.txt]" >&2
    exit 2
  fi
  summarize_file "$2" "${3:-}"
  exit $?
fi

if [[ $# -gt 0 ]]; then
  PACKAGES=("$@")
else
  # shellcheck disable=SC2206
  PACKAGES=(${SCOPE:-./...})
fi

mkdir -p "$PROFILE_DIR"
JSONL="$PROFILE_DIR/go-test.jsonl"
TIMING="$PROFILE_DIR/time.txt"
SUMMARY="$PROFILE_DIR/summary.txt"

# GO_TEST_FLAGS is intentionally unquoted so callers can pass multiple flags.
# Empty arrays are not expanded under set -u (bash 3.2 / stock macOS).
# shellcheck disable=SC2206
if [[ -n "${GO_TEST_FLAGS:-}" ]]; then
  EXTRA_FLAGS=(${GO_TEST_FLAGS})
else
  EXTRA_FLAGS=()
fi

echo "==> go test -json ${EXTRA_FLAGS[*]:-} ${PACKAGES[*]}"
echo "    jsonl: $JSONL"

run_go_test() {
  if [[ -x /usr/bin/time ]]; then
    /usr/bin/time -p -o "$TIMING" "$@"
  else
    echo "warn: /usr/bin/time not found; running without POSIX wall" >&2
    "$@"
  fi
}

set +e
if ((${#EXTRA_FLAGS[@]})); then
  run_go_test go test -json "${EXTRA_FLAGS[@]}" "${PACKAGES[@]}" >"$JSONL"
else
  run_go_test go test -json "${PACKAGES[@]}" >"$JSONL"
fi
ec=$?
set -e

# Status 1 from the summarizer is a completed failure table; still print EXIT.
# Status >1 is a summarizer parse error. Prefer the go test status when it already failed.
set +e
summarize_file "$JSONL" "$TIMING" | tee "$SUMMARY"
summary_ec=${PIPESTATUS[0]}
set -e
{
  echo
  echo "EXIT:$ec"
} | tee -a "$SUMMARY"
if (( summary_ec > 1 && ec == 0 )); then
  exit "$summary_ec"
fi

echo "==> done (profile dir: $PROFILE_DIR)"
exit "$ec"
