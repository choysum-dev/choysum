// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a navigable record id from a list row-click payload.
 * Accepts a raw model row or a `{ row }` wrapper; blank ids fail closed.
 */
export function resolveListRowRecordId(payload: unknown): string {
  if (!payload || typeof payload !== 'object') return '';
  const bag = payload as { row?: unknown; Id?: unknown };
  const row =
    bag.row && typeof bag.row === 'object' ? (bag.row as { Id?: unknown }) : bag;
  return String(row?.Id ?? '').trim();
}
