// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { SelectionItem } from './field';

/**
 * Merge static selection options by value.
 * Same value → later label (and labelText) wins; order = base order then new values.
 * Values must already be symbolic keys (non-empty, no leading/trailing whitespace).
 */
export function mergeSelectionByValue(base: SelectionItem[] | undefined, addends: SelectionItem[]): SelectionItem[] {
  const baseList = Array.isArray(base) ? base : [];
  const byValue = new Map<string, SelectionItem>();
  const order: string[] = [];

  function takeValue(raw: unknown): string | null {
    if (raw == null) return null;
    if (typeof raw !== 'string') {
      throw new Error('selection value must be a string');
    }
    if (raw !== raw.trim()) {
      throw new Error('selection value must not contain leading or trailing whitespace');
    }
    if (!raw) {
      throw new Error('selection value must be non-empty');
    }
    return raw;
  }

  for (const item of baseList) {
    if (!item || typeof item !== 'object') continue;
    const value = takeValue(item.value);
    if (value == null) continue;
    if (!byValue.has(value)) order.push(value);
    byValue.set(value, { ...item, value });
  }

  for (const item of addends) {
    if (!item || typeof item !== 'object') continue;
    const value = takeValue(item.value);
    if (value == null) continue;
    if (!byValue.has(value)) order.push(value);
    // Replace the whole option: a plain-string label must clear parent labelText
    // so FieldsGet does not keep translating the inherited term.
    byValue.set(value, {
      value,
      label: item.label,
      ...(item.labelText !== undefined ? { labelText: item.labelText } : {}),
    });
  }

  return order.map(value => byValue.get(value)!);
}
