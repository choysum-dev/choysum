// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/** Fixed pixel height, content-sized (`auto`), or any CSS length string. */
export type TreeBoxHeight = number | 'auto' | string;

const AUTO_STYLE: Record<string, string> = { height: 'auto' };

/**
 * Box style for M2M / M2MRef tree field hosts.
 * - `number`: fixed height in px (scrolls when content overflows)
 * - `'auto'` / empty / invalid: grow with content
 * - other string: used as CSS `height` (e.g. `20rem`, `50vh`)
 */
export function resolveTreeBoxStyle(treeHeight: TreeBoxHeight | undefined | null): Record<string, string> {
  if (treeHeight === 'auto') {
    return { ...AUTO_STYLE };
  }
  if (typeof treeHeight === 'number') {
    if (!Number.isFinite(treeHeight) || treeHeight <= 0) {
      return { ...AUTO_STYLE };
    }
    return { height: `${Math.floor(treeHeight)}px` };
  }
  const raw = String(treeHeight ?? '').trim();
  if (!raw || raw === 'auto') {
    return { ...AUTO_STYLE };
  }
  return { height: raw };
}
