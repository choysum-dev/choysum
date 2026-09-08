// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { afterLocaleChange, resolveLocaleRemountMode } from './locale_remount';

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

describe('afterLocaleChange', () => {
  test('defaults to reload', async () => {
    const reload = fnRecorder();
    await afterLocaleChange({ mode: 'reload', reload });
    expect(reload.calls.length).toBe(1);
  });

  test('calls remount hook when mode is remount', async () => {
    const remount = fnRecorder();
    const reload = fnRecorder();
    await afterLocaleChange({ mode: 'remount', remount, reload });
    expect(remount.calls.length).toBe(1);
    expect(reload.calls.length).toBe(0);
  });

  test('falls back to reload when remount mode has no hook', async () => {
    const reload = fnRecorder();
    await afterLocaleChange({ mode: 'remount', reload });
    expect(reload.calls.length).toBe(1);
  });

  test('resolveLocaleRemountMode defaults to reload', () => {
    expect(resolveLocaleRemountMode()).toBe('reload');
  });
});
