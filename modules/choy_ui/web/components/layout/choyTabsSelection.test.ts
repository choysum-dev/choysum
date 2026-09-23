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
});
