#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
# SPDX-License-Identifier: LGPL-3.0-or-later
#
# CI gate for terminology catalogs (S6).
# Exit non-zero when missing/fuzzy/orphan/pot-dirty issues exist.
# Disable or skip this step to leave runtime unaffected.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

LANG_CODE="${I18N_STATUS_LANG:-zh_CN}"
EXTRA_ARGS=()
if [[ "${I18N_STATUS_SKIP_POT:-0}" == "1" ]]; then
  EXTRA_ARGS+=(--skip-pot-check)
fi

if [[ -x ./choysum ]]; then
  ./choysum i18n status --all --lang "$LANG_CODE" "${EXTRA_ARGS[@]}"
else
  go run . i18n status --all --lang "$LANG_CODE" "${EXTRA_ARGS[@]}"
fi
