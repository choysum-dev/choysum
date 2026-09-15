#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
#
# Warm type-fetch assets into the CLI pkg cache and repair hollow vue graphs.
# Requires: CHOYSUM_BIN (path to choysum), CHOYSUM_TEST_TMP (from setup-cli-testing-cache),
# and modules/tsconfig.json writable in the workspace (gitignored; written by type-fetch).

set -euo pipefail

# type-fetch writes modules/tsconfig.json relative to the repo root.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../.."

if [[ -z "${CHOYSUM_BIN:-}" ]]; then
  echo "error: CHOYSUM_BIN is required" >&2
  exit 1
fi
if [[ ! -x "$CHOYSUM_BIN" ]]; then
  echo "error: CHOYSUM_BIN is not executable: $CHOYSUM_BIN" >&2
  exit 1
fi
if [[ -z "${CHOYSUM_TEST_TMP:-}" ]]; then
  echo "error: CHOYSUM_TEST_TMP is required (run setup-cli-testing-cache first)" >&2
  exit 1
fi

# Idempotent: cached packages are reused. Needed because restore-keys may
# return an ESM-only or otherwise incomplete pkg tree without types.
echo "Warming type-fetch cache via type-fetch --all"
"$CHOYSUM_BIN" type-fetch --all

# Force-repair vue: --all can keep hollow runtime-* siblings that Stat
# as present. Delete the package cache + entry so the next fetch
# re-materializes @vue/runtime-dom/core/reactivity.
types_dir="${CHOYSUM_TEST_TMP}/cache/pkg/types"
mkdir -p "$types_dir"
rm -f "$types_dir"/vue@*.d.ts \
  "$types_dir"/esm.sh_vue@* \
  "$types_dir"/esm.sh_@vue_runtime-dom@* \
  "$types_dir"/esm.sh_@vue_runtime-core@* \
  "$types_dir"/esm.sh_@vue_reactivity@*
echo "Re-fetching vue type graph"
"$CHOYSUM_BIN" type-fetch web

# Require the tsconfig-pinned vue entry (not a hollow/wrong-version graph).
# `|| true`: under pipefail, grep exit 1 on no match would skip the diagnostic below.
pinned="$(grep -oE 'esm\.sh_vue@[0-9][^/_"]+' modules/tsconfig.json | head -1 | sed 's/.*@//' || true)"
if [[ -z "$pinned" ]]; then
  echo "error: pinned vue version not found in modules/tsconfig.json" >&2
  exit 1
fi
echo "Pinned vue from tsconfig: $pinned"
vue_entry="$types_dir/esm.sh_vue@${pinned}_dist_vue.d.mts.d.ts"
core_dts="$types_dir/esm.sh_@vue_runtime-core@${pinned}_dist_runtime-core.d.ts.d.ts"
react_dts="$types_dir/esm.sh_@vue_reactivity@${pinned}_dist_reactivity.d.ts.d.ts"
if [[ ! -s "$vue_entry" || ! -s "$core_dts" || ! -s "$react_dts" ]]; then
  echo "error: pinned vue@$pinned type-fetch graph missing under $types_dir" >&2
  ls -la "$types_dir"/esm.sh_vue@* "$types_dir"/esm.sh_@vue_runtime-* "$types_dir"/esm.sh_@vue_reactivity@* 2>/dev/null || true
  exit 1
fi
if ! grep -Eq '\bPropType\b' "$core_dts"; then
  echo "error: runtime-core@$pinned missing PropType (hollow?)" >&2
  exit 1
fi
if ! grep -Eq '\btoRef\b' "$react_dts"; then
  echo "error: reactivity@$pinned missing toRef (hollow?)" >&2
  exit 1
fi
core_bytes="$(wc -c < "$core_dts" | tr -d ' ')"
if [[ "$core_bytes" -lt 10000 ]]; then
  echo "error: runtime-core@$pinned too small (${core_bytes} bytes); expected full .d.ts" >&2
  exit 1
fi
echo "Vue type-fetch graph OK (vue@$pinned, runtime-core ${core_bytes} bytes)"
node_bridge="${types_dir}/typeRoots/node/index.d.ts"
if [[ ! -s "$node_bridge" ]]; then
  echo "error: @types/node typeRoots bridge missing at $node_bridge" >&2
  ls -la "$types_dir"/typeRoots 2>/dev/null || true
  ls -la "$types_dir"/esm.sh_@types_node@* 2>/dev/null || true
  exit 1
fi
echo "Node typeRoots bridge OK ($node_bridge)"
