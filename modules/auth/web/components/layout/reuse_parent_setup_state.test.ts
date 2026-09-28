// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reuseParentSetupState } from './reuse_parent_setup_state.ts';

describe('reuseParentSetupState', () => {
  test('keeps plain object setup state', () => {
    expect(reuseParentSetupState({ showHeader: true, label: 'x' })).toEqual({
      showHeader: true,
      label: 'x',
    });
  });

  test('drops render functions and non-objects', () => {
    expect(reuseParentSetupState(() => null)).toEqual({});
    expect(reuseParentSetupState(null)).toEqual({});
    expect(reuseParentSetupState(undefined)).toEqual({});
    expect(reuseParentSetupState('state')).toEqual({});
    expect(reuseParentSetupState([{ a: 1 }])).toEqual({});
  });
});
