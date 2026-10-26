// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, computed, nextTick } from 'vue';
import { useListViewExpose } from './useListView';
import { asyncFnRecorder } from '@/web/web/__tests__/mountApp';

describe('useListViewExpose', () => {
  test('refresh delegates to the underlying list load', async () => {
    const load = asyncFnRecorder(async () => {});
    const { listRef, expose } = useListViewExpose<{ Id: string }>();
    listRef.value = { selectedItems: ref([]) as any, selectedItem: ref(null) as any, load } as any;
    await expose.refresh?.();
    expect(load.calls.length).toBe(1);
  });

  test('selectedItems tracks raw $.exposed Ref updates (Search-more OK path)', async () => {
    const { listRef, expose } = useListViewExpose<{ Id: string }>();
    const selectedItems = ref<{ Id: string }[]>([]);
    // Mimic Vue component public instance: unwrapped proxy reads can stale-cache;
    // expose must read $.exposed.selectedItems to stay live.
    listRef.value = {
      selectedItems: [],
      selectedItem: null,
      $: { exposed: { selectedItems, selectedItem: computed(() => selectedItems.value[0] ?? null) } },
      load: async () => {},
    } as any;

    expect(expose.selectedItems.value).toEqual([]);
    selectedItems.value = [{ Id: 'a' }, { Id: 'b' }];
    await nextTick();
    expect(expose.selectedItems.value).toEqual([{ Id: 'a' }, { Id: 'b' }]);
  });
});
