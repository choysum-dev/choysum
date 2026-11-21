// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Outer remount key for App-shell KeepAlive: company scope + leaf menu id.
 * Changing either fragment drops the entire cached instance table.
 */
export function keepAliveScopeKey(
  companyKey: unknown,
  menuId: unknown,
): string {
  const company = String(companyKey ?? '');
  const menu = String(menuId ?? '').trim();
  return `${company}::${menu}`;
}

/**
 * Keep the last non-empty leaf menu id so brief activeMenuId → null flaps
 * (cold-start sync) do not remount KeepAlive.
 */
export function stickyLeafMenuId(previous: string, next: unknown): string {
  const id = String(next ?? '').trim();
  return id !== '' ? id : previous;
}
