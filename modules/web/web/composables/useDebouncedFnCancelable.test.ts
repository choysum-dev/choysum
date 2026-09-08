// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { useDebouncedFnCancelable } from './useDebouncedFnCancelable';

function sleep(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms));
}

test('useDebouncedFnCancelable > debounces calls and only runs the latest scheduled call', async () => {
  const calls: number[] = [];
  const debounced = useDebouncedFnCancelable((n: number) => {
    calls.push(n);
  }, 40);

  debounced(1);
  debounced(2);
  debounced(3);

  expect(calls).toEqual([]);
  await sleep(25);
  expect(calls).toEqual([]);
  await sleep(30);
  expect(calls).toEqual([3]);
});

test('useDebouncedFnCancelable > cancel prevents the pending call from running', async () => {
  const calls: number[] = [];
  const debounced = useDebouncedFnCancelable((n: number) => calls.push(n), 40);

  debounced(1);
  debounced.cancel();

  await sleep(60);
  expect(calls).toEqual([]);
});
