// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { canShowAction } from './actionVisibility';

type CallRecorder = { calls: unknown[][] };

function fnRecorder<T = undefined, A extends unknown[] = unknown[]>(
  impl?: (...args: A) => T | Promise<T>
): CallRecorder & ((...args: A) => T | Promise<T>) {
  const rec: CallRecorder & ((...args: A) => T | Promise<T>) = Object.assign(
    (...args: A) => {
      rec.calls.push(args);
      return impl ? impl(...args) : (undefined as T);
    },
    { calls: [] as unknown[][] }
  );
  return rec;
}

describe('canShowAction', () => {
  test('returns true when action id is empty', () => {
    const checker = fnRecorder(() => false);
    expect(canShowAction(undefined, checker)).toBe(true);
    expect(canShowAction('', checker)).toBe(true);
    expect(checker.calls.length).toBe(0);
  });

  test('returns true when checker is missing', () => {
    expect(canShowAction('auth.action.user_create', undefined)).toBe(true);
  });

  test('delegates to checker when both are provided', () => {
    const checker = fnRecorder((id: string | undefined) => id === 'auth.action.user_create');
    expect(canShowAction('auth.action.user_create', checker)).toBe(true);
    expect(canShowAction('auth.action.user_delete', checker)).toBe(false);
    expect(checker.calls.length).toBe(2);
  });
});
