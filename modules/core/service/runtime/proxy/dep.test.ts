// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Dep } from './dep';

test('dep depend is no-op when no active target', () => {
  const dep = new Dep();
  const calls = { a: 0 };
  const watcher = {
    update: () => {
      calls.a += 1;
    },
  } as any;

  Dep.target = null;
  dep.depend();

  Dep.target = watcher;
  dep.depend();
  Dep.target = null;

  dep.notify();
  expect(calls.a).toBe(1);
});

test('dep tracks unique watchers and notifies all subscribers', () => {
  const dep = new Dep();
  const calls = { a: 0, b: 0 };
  const watcherA = {
    update: () => {
      calls.a += 1;
    },
  } as any;
  const watcherB = {
    update: () => {
      calls.b += 1;
    },
  } as any;

  Dep.target = watcherA;
  dep.depend();
  dep.depend();

  Dep.target = watcherB;
  dep.depend();

  Dep.target = null;
  dep.notify();

  expect(calls).toEqual({ a: 1, b: 1 });
});
