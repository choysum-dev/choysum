// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { MenuItem } from '@/core/web/menu';

export type AppNavGroup = {
  key: string;
  label: string;
  items: MenuItem[];
};

function visibleItems(list: MenuItem[] | undefined): MenuItem[] {
  return (list || []).filter((item) => !item.hidden);
}

/**
 * Groups an app root's children the same way ChoySidebarNav does:
 * section roots (with visible children) become labeled groups; loose leaves
 * sit in an unlabeled group at the front. A root with no children yields a
 * single ungrouped entry containing the root itself.
 */
export function buildAppNavGroups(
  root: MenuItem | null | undefined,
  labelOf: (item: MenuItem) => string,
): AppNavGroup[] {
  if (!root) return [];
  const children = visibleItems(root.children);
  if (!children.length) {
    return [{ key: 'ungrouped', label: '', items: [root] }];
  }

  const groups: AppNavGroup[] = [];
  const loose: MenuItem[] = [];
  for (const child of children) {
    const nested = visibleItems(child.children);
    if (nested.length) {
      groups.push({
        key: child.id || child.path || child.title,
        label: labelOf(child),
        items: nested,
      });
    } else {
      loose.push(child);
    }
  }
  if (loose.length) {
    groups.unshift({
      key: `${root.id || root.path || 'root'}-loose`,
      label: '',
      items: loose,
    });
  }
  return groups;
}

/** Visible non-hidden children of a menu node (for nested flyout rows). */
export function visibleMenuChildren(item: MenuItem | null | undefined): MenuItem[] {
  return visibleItems(item?.children);
}
