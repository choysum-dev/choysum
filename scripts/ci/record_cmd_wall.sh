#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
#
# Run a command under POSIX `time -p` and append real/user/sys to
# GITHUB_STEP_SUMMARY when that file is set (GitHub Actions). The wrapped
# command's stdout/stderr are left unchanged. Exit status is the command's.
#
# Usage:
#   ./scripts/ci/record_cmd_wall.sh go test ./... -count=1 ...

set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "usage: $0 <command> [args...]" >&2
  exit 2
fi

timing="$(mktemp)"
trap 'rm -f "$timing"' EXIT

set +e
/usr/bin/time -p -o "$timing" "$@"
ec=$?
set -e

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo "### go test wall"
    echo
    printf 'Command: `%s`\n' "$*"
    echo
    echo "CI measurement. Do not compare to a local \`go test ./...\` without \`-count=1\` / \`-cover\`."
    echo
    echo '```'
    cat "$timing"
    echo '```'
  } >>"$GITHUB_STEP_SUMMARY"
else
  echo "==> POSIX time:" >&2
  cat "$timing" >&2
fi

exit "$ec"
