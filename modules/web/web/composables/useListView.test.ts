// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { useListViewExpose } from './useListView';

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

describe('useListViewExpose', () => {
  test('refresh delegates to the underlying list load', async () => {
    const load = fnRecorder(async () => {});
    const { listRef, expose } = useListViewExpose<{ Id: string }>();
    listRef.value = { selectedItems: ref([]), selectedItem: ref(null), load };
    await expose.refresh?.();
    expect(load.calls.length).toBe(1);
  });
});
