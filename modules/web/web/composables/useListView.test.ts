// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
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
});
