// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { h, Comment, Fragment, Text } from 'vue';
import { resolveStatDisplayValue, slotHasContent } from './ostatinfo_helpers';

// density backfill from main before merge
// OStatInfo / OButtonBox mount suites need Element Plus icon stubs + fuller DOM under QJS.

describe('ostatinfo_helpers', () => {
  test('prefers explicit value over relation length', () => {
    expect(resolveStatDisplayValue({ value: 3, relationValue: [1, 2] })).toBe(3);
    expect(resolveStatDisplayValue({ value: 0, relationValue: [1] })).toBe(0);
  });

  test('uses relation array length when value is absent', () => {
    expect(resolveStatDisplayValue({ relationValue: ['a', 'b'] })).toBe(2);
    expect(resolveStatDisplayValue({ relationValue: [] })).toBe(0);
  });

  test('falls back to em dash (not 0) when unloaded', () => {
    expect(resolveStatDisplayValue({})).toBe('—');
    expect(resolveStatDisplayValue({ relationValue: null })).toBe('—');
    expect(resolveStatDisplayValue({ value: null })).toBe('—');
    expect(resolveStatDisplayValue({ emptyValue: 0 })).toBe(0);
  });

  test('detects empty vs meaningful slot trees', () => {
    expect(slotHasContent(null)).toBe(false);
    expect(slotHasContent(undefined)).toBe(false);
    expect(slotHasContent([])).toBe(false);
    expect(slotHasContent([h(Comment, 'x')])).toBe(false);
    expect(slotHasContent([h(Text, '   ')])).toBe(false);
    expect(slotHasContent([h(Text)])).toBe(false);
    expect(slotHasContent([h(Text, 'hi')])).toBe(true);
    expect(slotHasContent([h('div')])).toBe(true);
    expect(slotHasContent([h(Fragment, [h(Comment), h('span')])])).toBe(true);
    expect(slotHasContent([null, undefined, 42, 'plain'])).toBe(false);
    expect(slotHasContent([{ type: Fragment, children: 'x' }])).toBe(false);
  });
});

test('OStatInfo smoke: default export is a named Vue component', async () => {
  const mod = await import('./OStatInfo.vue');
  expect(mod.default).toBeTruthy();
  const name =
    (mod.default as { name?: string; __name?: string }).name ||
    (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});

test('OButtonBox smoke: default export is a named Vue component', async () => {
  const mod = await import('./OButtonBox.vue');
  expect(mod.default).toBeTruthy();
  const name =
    (mod.default as { name?: string; __name?: string }).name ||
    (mod.default as { name?: string; __name?: string }).__name;
  expect(name).toBeTruthy();
});
