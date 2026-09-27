<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Store-bound engine: host OListView (search, selection, OVColumn/ChoyVColumn slots). -->
  <OListView v-if="useStoreEngine" v-bind="(storeBind as any)" v-on="(storeListeners as any)">
    <slot />
  </OListView>

  <!-- Chrome: host-supplied columns + data (Gallery / Dogfood). -->
  <div
    v-else
    data-anchor="choy.list-view"
    :class="['choy-list-view flex w-full flex-col gap-3', props.class]"
  >
    <div
      v-if="$slots.header || $slots.search"
      class="choy-list-view__toolbar flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
    >
      <div v-if="$slots.header" class="choy-list-view__header min-w-0 flex-1">
        <slot name="header" />
      </div>
      <div v-if="$slots.search" class="choy-list-view__search shrink-0">
        <slot name="search" />
      </div>
    </div>
    <DataTable
      :columns="columns ?? []"
      :data="data ?? []"
      :row-id="rowId"
      :row-selection="rowSelection"
      :height="height"
      :enable-row-selection="enableRowSelection"
      :enable-sorting="enableSorting"
      @update:row-selection="onRowSelection"
      @row-click="onRowClick"
    />
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, unknown>">
import { computed, useAttrs } from 'vue';
import type { ColumnDef } from '@tanstack/vue-table';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useOptionalPageStore } from '@/web/web/composables/usePageContext';
import { hasChoyStoreEngine } from '@/web/web/composables/choyStoreMode';
import DataTable from '../internal/DataTable.vue';
import type { DataTableRowId } from '../internal/dataTableHelpers';
import type { ClassValue } from '../../lib/utils';
import OListView from './OListView.vue';

/**
 * List view: store-bound mode hosts OListView; otherwise DataTable chrome.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    columns?: ColumnDef<T, unknown>[];
    data?: T[];
    rowId?: (row: T) => DataTableRowId;
    height?: number;
    enableRowSelection?: boolean;
    enableSorting?: boolean;
    store?: WebModelStore<any>;
  }>(),
  {
    height: 280,
    enableRowSelection: true,
    enableSorting: true,
  },
);

const attrs = useAttrs();
const pageStore = useOptionalPageStore();
const useStoreEngine = computed(() => hasChoyStoreEngine(props.store, pageStore.value));

const storeBind = computed(() => ({
  ...attrs,
  store: props.store ?? pageStore.value,
  class: props.class,
}));

const storeListeners = computed(() => {
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(attrs)) {
    if (key.startsWith('on') && typeof value === 'function') {
      out[key] = value;
    }
  }
  return out;
});

const rowSelection = defineModel<DataTableRowId[]>('rowSelection', {
  default: () => [],
});

const emit = defineEmits<{
  'row-click': [row: T];
}>();

function onRowSelection(ids: DataTableRowId[]): void {
  rowSelection.value = ids;
}

function onRowClick(row: T): void {
  emit('row-click', row);
}
</script>
