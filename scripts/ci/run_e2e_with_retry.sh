#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
#
# Run `choysum test e2e` with a small retry budget for flake isolation.
# Requires GNU coreutils `timeout` (GitHub ubuntu-latest runners).
# Env:
#   CHOYSUM_BIN, MODULE (required)
#   E2E_EXTRA_ARGS       optional whitespace-separated CLI args after MODULE
#   E2E_MAX_ATTEMPTS     positive integer, default 2
#   E2E_ATTEMPT_TIMEOUT  GNU timeout duration per attempt, default 30m

set -euo pipefail

if [[ -z "${CHOYSUM_BIN:-}" || -z "${MODULE:-}" ]]; then
  echo "error: CHOYSUM_BIN and MODULE are required" >&2
  exit 1
fi
# MODULE comes from modules/<name> and is echoed into workflow commands; reject unsafe names.
if [[ ! "$MODULE" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "error: invalid MODULE name: ${MODULE}" >&2
  exit 1
fi
if [[ ! -x "$CHOYSUM_BIN" ]]; then
  echo "error: CHOYSUM_BIN is not executable: $CHOYSUM_BIN" >&2
  exit 1
fi

max_attempts="${E2E_MAX_ATTEMPTS:-2}"
if ! [[ "$max_attempts" =~ ^[1-9][0-9]*$ ]]; then
  echo "error: E2E_MAX_ATTEMPTS must be a positive integer, got '${max_attempts}'" >&2
  exit 1
fi

attempt_timeout="${E2E_ATTEMPT_TIMEOUT:-30m}"
if ! [[ "$attempt_timeout" =~ ^[1-9][0-9]*[smhd]?$ ]]; then
  echo "error: E2E_ATTEMPT_TIMEOUT must be a duration like 30m, got '${attempt_timeout}'" >&2
  exit 1
fi

extra_args=()
if [[ -n "${E2E_EXTRA_ARGS:-}" ]]; then
  # Split on whitespace without pathname glob expansion.
  read -r -a extra_args <<< "${E2E_EXTRA_ARGS}"
fi

attempt=1
while true; do
  echo "E2E attempt ${attempt}/${max_attempts} for module=${MODULE} (timeout=${attempt_timeout})"
  set +e
  # Bound each attempt so a hung Chromium run cannot consume the whole job timeout.
  # ${arr[@]+"${arr[@]}"} avoids set -u unbound-array on empty extra_args (bash <4.4).
  timeout --signal=TERM --kill-after=30s "${attempt_timeout}" \
    "$CHOYSUM_BIN" test e2e "$MODULE" ${extra_args[@]+"${extra_args[@]}"}
  exit_code=$?
  set -e
  if [[ "$exit_code" -eq 0 ]]; then
    exit 0
  fi
  # Setup / misconfig exits: do not retry as flakes.
  if [[ "$exit_code" -eq 125 || "$exit_code" -eq 126 || "$exit_code" -eq 127 ]]; then
    echo "error: E2E setup/misconfiguration (exit=${exit_code}); not retrying" >&2
    exit "$exit_code"
  fi
  if [[ "$attempt" -ge "$max_attempts" ]]; then
    echo "E2E failed after ${max_attempts} attempt(s) (exit=${exit_code})" >&2
    exit "$exit_code"
  fi
  reason="assertion-or-runtime"
  if [[ "$exit_code" -eq 124 || "$exit_code" -eq 137 ]]; then
    reason="attempt-timeout"
  fi
  echo "E2E attempt ${attempt} failed (exit=${exit_code}, reason=${reason}); retrying..." >&2
  echo "::warning title=E2E ${reason}::module=${MODULE} attempt ${attempt}/${max_attempts} failed (exit=${exit_code}); retrying" || true
  attempt=$((attempt + 1))
  sleep 5
done
