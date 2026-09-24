<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts" generic="T extends Record<string, unknown>">
import type { ColumnDef } from '@tanstack/vue-table';
import DataTable from '../internal/DataTable.vue';
import type { DataTableRowId } from '../internal/dataTableHelpers';
import type { ClassValue } from '../../lib/utils';

/**
 * List view chrome wrapping L3 DataTable. Optional header/search toolbar slots.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    columns: ColumnDef<T, unknown>[];
    data: T[];
    rowId?: (row: T) => DataTableRowId;
    height?: number;
    enableRowSelection?: boolean;
    enableSorting?: boolean;
  }>(),
  {
    height: 280,
    enableRowSelection: true,
    enableSorting: true,
  },
);

const rowSelection = defineModel<DataTableRowId[] | null>('rowSelection', {
  default: null,
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

<template>
  <div
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
      :columns="columns"
      :data="data"
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
