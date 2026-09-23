// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { createChoyTabsContext } from './choyTabsContext';

describe('createChoyTabsContext', () => {
  test('refuses duplicate register and keeps order on in-place rename', () => {
    const ctx = createChoyTabsContext();
    expect(
      ctx.register({ value: 'a', label: 'A', disabled: false }),
    ).toBe(true);
    expect(
      ctx.register({ value: 'b', label: 'B', disabled: false }),
    ).toBe(true);
    expect(
      ctx.register({ value: 'a', label: 'Dup', disabled: false }),
    ).toBe(false);

    expect(ctx.update('a', { value: 'b', label: 'A2' })).toBe(false);
    expect(ctx.update('a', { value: 'c', label: 'C', disabled: true })).toBe(true);
    expect(ctx.tabs.value.map((t) => t.value)).toEqual(['c', 'b']);
    expect(ctx.tabs.value[0]).toEqual({ value: 'c', label: 'C', disabled: true });
  });

  test('unregister removes only the named registration', () => {
    const ctx = createChoyTabsContext();
    ctx.register({ value: 'a', label: 'A', disabled: false });
    ctx.register({ value: 'b', label: 'B', disabled: false });
    ctx.unregister('a');
    expect(ctx.tabs.value.map((t) => t.value)).toEqual(['b']);
  });

  test('update / unregister on an unknown value are non-destructive', () => {
    const ctx = createChoyTabsContext();
    ctx.register({ value: 'a', label: 'A', disabled: false });
    expect(ctx.update('missing', { label: 'X' })).toBe(false);
    ctx.unregister('missing');
    expect(ctx.tabs.value).toEqual([{ value: 'a', label: 'A', disabled: false }]);
  });

  test('update with undefined value in the patch keeps tab identity', () => {
    const ctx = createChoyTabsContext();
    ctx.register({ value: 'a', label: 'A', disabled: false });
    expect(ctx.update('a', { value: undefined, label: 'Renamed' })).toBe(true);
    expect(ctx.tabs.value).toEqual([{ value: 'a', label: 'Renamed', disabled: false }]);
  });
});
