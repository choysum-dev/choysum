// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { SelectionItem } from './field';

/**
 * Merge static selection options by value (PR-P2-F4 / D4).
 * Same value → later label (and labelText) wins; order = base order then new values.
 */
export function mergeSelectionByValue(base: SelectionItem[] | undefined, addends: SelectionItem[]): SelectionItem[] {
  const baseList = Array.isArray(base) ? base : [];
  const byValue = new Map<string, SelectionItem>();
  const order: string[] = [];

  for (const item of baseList) {
    const value = String(item?.value ?? '').trim();
    if (!value) continue;
    if (!byValue.has(value)) order.push(value);
    byValue.set(value, { ...item, value });
  }

  for (const item of addends) {
    const value = String(item?.value ?? '').trim();
    if (!value) continue;
    if (!byValue.has(value)) order.push(value);
    const prev = byValue.get(value);
    byValue.set(value, {
      value,
      label: item.label,
      ...(item.labelText !== undefined
        ? { labelText: item.labelText }
        : prev?.labelText !== undefined
          ? { labelText: prev.labelText }
          : {}),
    });
  }

  return order.map(value => byValue.get(value)!);
}
