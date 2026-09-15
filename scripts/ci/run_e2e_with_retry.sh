#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
#
# Run `choysum test e2e` with a small retry budget for flake isolation.
# Env: CHOYSUM_BIN, MODULE, optional E2E_EXTRA_ARGS, E2E_MAX_ATTEMPTS (default 2).

set -euo pipefail

if [[ -z "${CHOYSUM_BIN:-}" || -z "${MODULE:-}" ]]; then
  echo "error: CHOYSUM_BIN and MODULE are required" >&2
  exit 1
fi

max_attempts="${E2E_MAX_ATTEMPTS:-2}"
extra_args=()
# shellcheck disable=SC2206
if [[ -n "${E2E_EXTRA_ARGS:-}" ]]; then
  # Intentionally unquoted: callers pass a space-separated flag string.
  extra_args=(${E2E_EXTRA_ARGS})
fi

attempt=1
while true; do
  echo "E2E attempt ${attempt}/${max_attempts} for module=${MODULE}"
  set +e
  "$CHOYSUM_BIN" test e2e "$MODULE" "${extra_args[@]}"
  exit_code=$?
  set -e
  if [[ "$exit_code" -eq 0 ]]; then
    exit 0
  fi
  if [[ "$attempt" -ge "$max_attempts" ]]; then
    echo "E2E failed after ${max_attempts} attempt(s) (exit=${exit_code})" >&2
    exit "$exit_code"
  fi
  echo "E2E attempt ${attempt} failed (exit=${exit_code}); retrying..." >&2
  attempt=$((attempt + 1))
  sleep 5
done
