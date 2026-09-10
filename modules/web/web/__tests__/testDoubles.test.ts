// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import {
  asyncFnRecorder,
  emptyAsyncIterable,
  fnRecorder,
  syncFnRecorder,
} from './testDoubles';
import { mountApp } from './mountApp';
import { defineComponent, h, type Plugin } from 'vue';

test('fnRecorder records calls and supports mock helpers', () => {
  const rec = fnRecorder((n: number) => n * 2);
  expect(rec(3)).toBe(6);
  expect(rec.calls).toEqual([[3]]);
  rec.mockClear();
  expect(rec.calls).toEqual([]);
  rec.mockReturnValue(9);
  expect(rec(1)).toBe(9);
  rec.mockImplementation((n: number) => n + 1);
  expect(rec(4)).toBe(5);
  rec.mockReset();
  expect(rec(5)).toBe(10);
});

test('syncFnRecorder delegates to fnRecorder', () => {
  const rec = syncFnRecorder(() => 'ok');
  expect(rec()).toBe('ok');
  expect(rec.calls).toHaveLength(1);
});

test('asyncFnRecorder always returns a Promise and rejects sync throws', async () => {
  const empty = asyncFnRecorder();
  expect(await empty()).toBeUndefined();

  const rec = asyncFnRecorder(() => 1);
  expect(await rec()).toBe(1);
  expect(rec.calls).toEqual([[]]);

  rec.mockReturnValue(2);
  expect(await rec('a')).toBe(2);

  rec.mockImplementation(() => {
    throw new Error('boom');
  });
  let rejected: unknown;
  try {
    await rec();
  } catch (err) {
    rejected = err;
  }
  expect(rejected).toBeInstanceOf(Error);
  expect(String((rejected as Error).message)).toContain('boom');
  expect(rec.calls.length).toBeGreaterThan(0);

  rec.mockClear();
  expect(rec.calls).toEqual([]);
  rec.mockReset();
  expect(await rec()).toBe(1);
});

test('emptyAsyncIterable yields done immediately', async () => {
  const it = emptyAsyncIterable<number>();
  const next = await it[Symbol.asyncIterator]().next();
  expect(next.done).toBe(true);
});

test('mountApp accepts plugin tuples with options', () => {
  let seen: unknown;
  let plainInstalled = false;
  const plugin: Plugin = {
    install(_app, opts) {
      seen = opts;
    },
  };
  const plain: Plugin = {
    install() {
      plainInstalled = true;
    },
  };
  const Comp = defineComponent({
    setup() {
      return () => h('div', { 'data-host': '1' });
    },
  });
  const mounted = mountApp(Comp, { plugins: [plain, [plugin, { flag: true }]] });
  expect(plainInstalled).toBe(true);
  expect(seen).toEqual({ flag: true });
  expect(mounted.q('[data-host]')).toBeTruthy();
  mounted.unmount();
});
