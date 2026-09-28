// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reuseParentSetupState } from './reuse_parent_setup_state.ts';

describe('reuseParentSetupState', () => {
  test('keeps plain object setup state', () => {
    expect(reuseParentSetupState({ showHeader: true, label: 'x' })).toEqual({
      showHeader: true,
      label: 'x',
    });
    const nullProto: Record<string, unknown> = Object.create(null);
    nullProto.showHeader = true;
    expect(reuseParentSetupState(nullProto)).toEqual({ showHeader: true });
  });

  test('drops render functions and non-objects', () => {
    expect(reuseParentSetupState(() => null)).toEqual({});
    expect(reuseParentSetupState(null)).toEqual({});
    expect(reuseParentSetupState(undefined)).toEqual({});
    expect(reuseParentSetupState('state')).toEqual({});
    expect(reuseParentSetupState([{ a: 1 }])).toEqual({});
  });

  test('drops non-plain objects such as Map and class instances', () => {
    expect(reuseParentSetupState(new Map([['a', 1]]))).toEqual({});
    class SetupBag {
      showHeader = true;
    }
    expect(reuseParentSetupState(new SetupBag())).toEqual({});
  });

  test('throws when parent setup returns a Promise', () => {
    expect(() => reuseParentSetupState(Promise.resolve({ showHeader: true }))).toThrow(
      /async parent setup is not supported/
    );
  });
});
