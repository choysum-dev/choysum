// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Concise O2M/M2M count text for List/Kanban cells (not Form tables).
 * Empty count uses a literal em dash (punctuation, not an i18n message).
 */
export function formatRelationCountSummary(
  count: number,
  _t: (msg: string, ...args: unknown[]) => string
): string {
  const n = Number(count) || 0;
  if (n <= 0) return '—';
  return n === 1 ? _t('%s record', n) : _t('%s records', n);
}
