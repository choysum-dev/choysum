// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive, readonly } from 'vue';
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

  test('keeps objects whose prototype chain terminates immediately', () => {
    // Simulate another realm's Object.prototype (not === local Object.prototype).
    const foreignObjectProto = Object.create(null);
    const foreignPlain: Record<string, unknown> = Object.create(foreignObjectProto);
    foreignPlain.showHeader = true;
    expect(reuseParentSetupState(foreignPlain)).toEqual({ showHeader: true });
  });

  test('returns a shallow copy that does not alias the parent object', () => {
    const parent = { showHeader: true };
    const reused = reuseParentSetupState(parent);
    reused.showHeader = false;
    expect(parent.showHeader).toBe(true);
    expect(reused).not.toBe(parent);
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

  test('drops reactive and readonly proxies', () => {
    expect(reuseParentSetupState(reactive({ showHeader: true }))).toEqual({});
    expect(reuseParentSetupState(readonly({ showHeader: true }))).toEqual({});
  });

  test('throws when parent setup returns a Promise or thenable', () => {
    expect(() => reuseParentSetupState(Promise.resolve({ showHeader: true }))).toThrow(
      /async parent setup is not supported/
    );
    const thenable = {
      showHeader: true,
      then(resolve: (v: unknown) => void) {
        resolve({ showHeader: true });
      },
    };
    expect(() => reuseParentSetupState(thenable)).toThrow(/async parent setup is not supported/);
  });
});
