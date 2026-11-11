// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Stable router-view key fragment for the current company scope.
 * Only active + sorted enabled ids; ignores token refresh noise.
 */
export function companyScopeViewKey(
  activeCompanyId: unknown,
  enabledCompanyIds: unknown,
): string {
  const active = String(activeCompanyId ?? '').trim();
  const enabled = Array.isArray(enabledCompanyIds)
    ? enabledCompanyIds.map(x => String(x ?? '').trim()).filter(Boolean)
    : [];
  const unique = Array.from(new Set(enabled)).sort();
  if (active && !unique.includes(active)) unique.unshift(active);
  unique.sort();
  return `${active}|${unique.join(',')}`;
}
