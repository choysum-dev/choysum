// @vitest-environment happy-dom
// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest';
import { ref } from 'vue';
import { useListViewExpose } from './useListView';

describe('useListViewExpose', () => {
  it('refresh delegates to the underlying list load', async () => {
    const load = vi.fn(async () => {});
    const { listRef, expose } = useListViewExpose<{ Id: string }>();
    listRef.value = { selectedItems: ref([]), selectedItem: ref(null), load };
    await expose.refresh?.();
    expect(load).toHaveBeenCalled();
  });
});
