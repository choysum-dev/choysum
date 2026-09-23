// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextChoyTabSelection, pickChoyTabSelection } from './choyTabsSelection';
import type { ChoyTabRegistration } from './choyTabsContext';

describe('choyTabsSelection', () => {
  test('waits for late-registering defaultValue instead of locking the first tab', () => {
    const firstOnly: ChoyTabRegistration[] = [
      { value: 'one', label: 'One', disabled: false },
    ];
    expect(nextChoyTabSelection(firstOnly, undefined, 'two')).toBeUndefined();
    expect(nextChoyTabSelection(firstOnly, undefined, 'two', true)).toBe('one');

    const both: ChoyTabRegistration[] = [
      ...firstOnly,
      { value: 'two', label: 'Two', disabled: false },
    ];
    expect(nextChoyTabSelection(both, undefined, 'two')).toBe('two');
  });

  test('does not select a disabled-only registration list', () => {
    const allDisabled: ChoyTabRegistration[] = [
      { value: 'one', label: 'One', disabled: true },
      { value: 'two', label: 'Two', disabled: true },
    ];
    expect(pickChoyTabSelection(allDisabled, 'one')).toBe('');
    expect(nextChoyTabSelection(allDisabled, undefined, undefined)).toBe('');
  });

  test('keeps a still-valid current selection', () => {
    const list: ChoyTabRegistration[] = [
      { value: 'one', label: 'One', disabled: false },
      { value: 'two', label: 'Two', disabled: false },
    ];
    expect(nextChoyTabSelection(list, 'two', 'one')).toBeUndefined();
  });

  test('falls back to the first enabled tab when the default is disabled', () => {
    const list: ChoyTabRegistration[] = [
      { value: 'one', label: 'One', disabled: false },
      { value: 'two', label: 'Two', disabled: true },
    ];
    expect(nextChoyTabSelection(list, undefined, 'two')).toBe('one');
  });

  test('rescues a stale selection when the default never registers', () => {
    const list: ChoyTabRegistration[] = [
      { value: 'one', label: 'One', disabled: false },
    ];
    expect(nextChoyTabSelection(list, 'gone', 'missing')).toBe('one');
    // Unsettled + empty current: keep waiting for the default.
    expect(nextChoyTabSelection(list, undefined, 'missing')).toBeUndefined();
    // Settled: fall back so a typo cannot leave the host with no selection.
    expect(nextChoyTabSelection(list, undefined, 'missing', true)).toBe('one');
  });

  test('leaves the selection untouched while no tabs are registered', () => {
    expect(nextChoyTabSelection([], undefined, undefined)).toBeUndefined();
    expect(nextChoyTabSelection([], 'one', 'one')).toBeUndefined();
  });

  test('recovers to the first enabled tab after an all-disabled list', () => {
    const disabledOnly: ChoyTabRegistration[] = [
      { value: 'one', label: 'One', disabled: true },
    ];
    expect(nextChoyTabSelection(disabledOnly, undefined, 'one')).toBe('');

    const enabled: ChoyTabRegistration[] = [
      { value: 'one', label: 'One', disabled: false },
    ];
    expect(nextChoyTabSelection(enabled, '', undefined)).toBe('one');
  });
});
