// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref, computed, unref, isRef } from 'vue';
import type { ListViewLoadExpose } from '@/web/web/components/view/listViewTypes';

/**
 * Read a child ListView selection expose.
 * Prefer `instance.$.exposed.*` (raw Ref) so the computed tracks updates; reading the
 * auto-unwrapped public proxy can cache a stale empty array and break Search-more OK.
 */
function readExposedSelection<T>(
  inst: unknown,
  key: 'selectedItems' | 'selectedItem',
): unknown {
  if (!inst) return undefined;
  const rawExposed = (inst as any)?.$?.exposed?.[key];
  if (rawExposed !== undefined) return unref(rawExposed);
  const pub = (inst as any)?.[key];
  return isRef(pub) ? unref(pub) : pub;
}

/**
 * Creates the ListView ref and the object exposed through defineExpose.
 * @template T Model type.
 */
export function useListViewExpose<T>() {
  // Create the ref bound to <ListView ref="listRef">.
  const listRef = ref<unknown>(null);

  const selectedItems = computed<T[]>(() => {
    const v = readExposedSelection<T>(listRef.value, 'selectedItems');
    return Array.isArray(v) ? (v as T[]) : [];
  });
  const selectedItem = computed<T | null>(() => {
    const v = readExposedSelection<T>(listRef.value, 'selectedItem');
    return (v as T | null | undefined) ?? null;
  });

  // Return both the ref and the object intended for defineExpose.
  function refresh() {
    return (listRef.value as ListViewLoadExpose | null | undefined)?.load?.();
  }

  return {
    listRef,
    expose: {
      selectedItems,
      selectedItem,
      refresh,
    },
  };
}
