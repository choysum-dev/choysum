// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, computed } from 'vue';
import type { SelectionExpose } from '@/web/web/components/view/OListView.vue';

/**
 * Creates the ListView ref and the object exposed through defineExpose.
 * @template T Model type.
 */
export function useListViewExpose<T>() {
  // Create the ref bound to <OListView ref="listRef">.
  const listRef = ref<SelectionExpose<T> | null>(null);

  // Proxy computed values that already handle null checks.
  const selectedItems = computed<T[]>(() => listRef.value?.selectedItems ?? []);
  const selectedItem = computed<T | null>(() => listRef.value?.selectedItem ?? null);

  // Return both the ref and the object intended for defineExpose.
  return {
    listRef,
    expose: {
      selectedItems,
      selectedItem,
    },
  };
}
