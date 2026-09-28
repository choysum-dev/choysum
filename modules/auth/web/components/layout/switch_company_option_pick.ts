// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Choose which company option the switch-company e2e helper should select.
 *
 * Prefers an already-selected draft that differs from the JWT active company
 * (retries after refreshToken can leave the select on a valid alternative).
 * Otherwise picks any option different from both current and active.
 */
export function pickAlternativeCompanyOptionValue(
  values: string[],
  current: string,
  active: string
): string {
  const cur = String(current || '').trim();
  const act = String(active || '').trim();
  const opts = values.map(v => String(v || '').trim()).filter(Boolean);
  if (cur && cur !== act && opts.includes(cur)) {
    return cur;
  }
  return opts.find(v => v !== cur && v !== act) || '';
}
