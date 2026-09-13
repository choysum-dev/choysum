#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
#
# Profile choysum install / uninstall / upgrade wall-clock and step duration_ms.
#
# Usage:
#   ./scripts/dev/lifecycle_op_profile.sh [module]
#
# Env:
#   MODULE                 default: partner
#   PROFILE_DIR            default: /tmp/choysum-profile
#   SKIP_BUILD=1           skip `go build -o choysum .`
#   CHOYSUM_BIN            path to choysum binary (default: ./choysum)
#   OPS                    space-separated ops (default: "upgrade uninstall install")
#
# Tip: source .envrc (or direnv) first; set CHOYSUM_LOG_LEVEL=debug for semantic metrics.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

MODULE="${1:-${MODULE:-partner}}"
PROFILE_DIR="${PROFILE_DIR:-/tmp/choysum-profile}"
CHOYSUM_BIN="${CHOYSUM_BIN:-$ROOT/choysum}"
OPS="${OPS:-upgrade uninstall install}"

mkdir -p "$PROFILE_DIR"
export CHOYSUM_LOG_LEVEL="${CHOYSUM_LOG_LEVEL:-info}"

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
  echo "==> go build -o choysum ."
  go build -o "$CHOYSUM_BIN" .
fi

summarize() {
  local log="$1"
  python3 - "$log" <<'PY'
import re, sys
from collections import defaultdict
path = sys.argv[1]
by = defaultdict(list)
walls = []
for line in open(path, errors="replace"):
    if line.startswith(("real ", "user ", "sys ", "EXIT:")):
        walls.append(line.strip())
        continue
    m = re.search(r'duration_ms=(\d+)', line)
    if not m:
        continue
    step = "-"
    if sm := re.search(r'\bstep=(\S+)', line):
        step = sm.group(1)
    msg = "?"
    if mm := re.search(r'msg="([^"]+)"', line):
        msg = mm.group(1)
    # Prefer step-keyed rows; keep interesting aggregates.
    key = step if step != "-" else msg
    if step == "-" and "duration_ms" in line and not any(
        k in msg for k in ("step completed", "transaction hold", "web built", "phase end", "entity migration", "semantic program", "operation completed", "operation plan")
    ):
        continue
    by[key].append(int(m.group(1)))
print(f"--- {path} ---")
for w in walls:
    print(w)
print("top duration_ms:")
rows = sorted(((max(v), sum(v), len(v), k) for k, v in by.items()), reverse=True)
for mx, sm, n, k in rows[:25]:
    if mx < 20 and not str(k).startswith(("hook.", "scripts.", "web_", "phase_", "base_")):
        continue
    print(f"  {mx:8d}ms  n={n:2d}  sum={sm:8d}  {k}")
PY
}

ec=0
for op in $OPS; do
  name="${op}_${MODULE}"
  log="$PROFILE_DIR/${name}.log"
  echo "==> $op $MODULE  ($(date +%H:%M:%S))"
  set +e
  /usr/bin/time -p "$CHOYSUM_BIN" "$op" "$MODULE" >"$log" 2>&1
  op_ec=$?
  set -e
  echo "EXIT:$op_ec" | tee -a "$log" >/dev/null
  summarize "$log"
  if [[ $op_ec -ne 0 ]]; then
    echo "!! $op failed (exit $op_ec); see $log"
    ec=$op_ec
  fi
done

echo "==> done (profile dir: $PROFILE_DIR)"
exit "$ec"
