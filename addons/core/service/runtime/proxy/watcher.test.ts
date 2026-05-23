// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { Dep } from './dep';
import { Watcher } from './watcher';

test('watcher lazily computes value and caches until update', () => {
  let calls = 0;
  const state = { n: 2 };
  const watcher = new Watcher(
    state,
    ({ self }) => {
      calls += 1;
      return self.n * 2;
    },
    'double'
  );

  expect(watcher.isInitialized()).toBe(false);
  expect(watcher.get()).toBe(4);
  expect(watcher.isInitialized()).toBe(true);
  expect(watcher.get()).toBe(4);
  expect(calls).toBe(1);

  state.n = 3;
  watcher.update();
  expect(watcher.get()).toBe(6);
  expect(calls).toBe(2);
  expect(watcher.getError()).toBe(null);
  expect(watcher.getComputeTime()).toBeGreaterThanOrEqual(0);
});

test('watcher captures errors and triggers onError callback', () => {
  let onErrorCount = 0;
  const boom = new Error('boom');
  const watcher = new Watcher(
    {},
    () => {
      throw boom;
    },
    'broken',
    () => {
      onErrorCount += 1;
    }
  );

  let message = '';
  try {
    watcher.get();
  } catch (error) {
    message = String((error as Error).message);
  }

  expect(message).toContain('boom');
  expect(onErrorCount).toBe(1);
  expect(watcher.getError()).toBe(boom);
  expect(Dep.target).toBe(null);
});

test('watcher throws on circular dependency and clears eval stack', () => {
  let selfWatcher: Watcher<any>;
  selfWatcher = new Watcher({}, () => selfWatcher.get(), 'self-loop');

  let message = '';
  try {
    selfWatcher.get();
  } catch (error) {
    message = String((error as Error).message);
  }

  expect(message).toContain('Detected circular computed property dependency');
  expect(message).toContain('self-loop');
  expect(Watcher.evalStack.length).toBe(0);
  expect(Dep.target).toBe(null);
});
