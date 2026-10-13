// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';

export type AppNavGroup = {
  key: string;
  /** Reserved; always empty — branch nodes stay as Collapsible / flyout Sub items. */
  label: string;
  items: MenuItem[];
};

function visibleItems(list: MenuItem[] | undefined): MenuItem[] {
  return (list || []).filter((item) => !item.hidden);
}

/**
 * Flattens an app root into one unlabeled nav group whose items keep their
 * nested children. Branch nodes render as Collapsible (sidebar) or DropdownMenuSub
 * (rail flyout) at any depth. A root with no children yields a single entry
 * containing the root itself.
 */
export function buildAppNavGroups(root: MenuItem | null | undefined): AppNavGroup[] {
  if (!root) return [];
  const declared = root.children || [];
  const children = visibleItems(declared);
  if (!children.length) {
    // Declared children that are all hidden → empty nav. No children at all →
    // treat the root itself as the sole leaf (path-only app).
    if (declared.length > 0) return [];
    return [{ key: 'ungrouped', label: '', items: [root] }];
  }
  return [
    {
      key: root.id || root.path || 'root',
      label: '',
      items: children,
    },
  ];
}

/** Visible non-hidden children of a menu node (for nested flyout rows). */
export function visibleMenuChildren(item: MenuItem | null | undefined): MenuItem[] {
  return visibleItems(item?.children);
}
