// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';

export type HomeShortcut = { id: string; label: string; path?: string };

/**
 * Collects up to `limit` internal menu shortcuts, skipping hidden/disabled
 * items, non-root-relative paths, and duplicate paths.
 */
export function collectHomeShortcuts(
  items: MenuItem[],
  limit: number,
  labelOf: (item: MenuItem) => string,
): HomeShortcut[] {
  const out: HomeShortcut[] = [];
  collect(items, out, limit, labelOf, new Set<MenuItem>());
  return out;
}

function collect(
  items: MenuItem[],
  out: HomeShortcut[],
  limit: number,
  labelOf: (item: MenuItem) => string,
  seen: Set<MenuItem>,
): void {
  for (const item of items) {
    if (out.length >= limit) return;
    if (!item || seen.has(item)) continue;
    seen.add(item);
    if (item.hidden || item.disabled) continue;
    if (
      typeof item.path === 'string' &&
      item.path.startsWith('/') &&
      !item.externalLink &&
      !out.some(entry => entry.path === item.path)
    ) {
      out.push({
        id: item.id || item.path,
        label: labelOf(item),
        path: item.path,
      });
    }
    if (item.children?.length) collect(item.children, out, limit, labelOf, seen);
  }
}
