<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-vtable" :style="{ height: tableHeight ? `${tableHeight}px` : undefined }">
    <!-- Hidden slot used only for column registration. -->
    <div class="ovtable__registrars" aria-hidden="true">
      <slot />
    </div>

    <div class="ovtable__scroll">
      <table class="ovtable__table">
        <thead>
          <tr :style="{ height: `${headerHeight}px` }">
            <th
              v-for="col in columnsToUse"
              :key="String(col.key ?? col.dataKey)"
              :style="thStyle(col)"
              @click="onHeaderClick(col)"
            >
              <component :is="() => renderHeader(col)" />
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="!rowsArray.length">
            <td :colspan="Math.max(columnsToUse.length, 1)">
              <slot name="empty">
                <div class="ovtable__empty">{{ _t('No data') }}</div>
              </slot>
            </td>
          </tr>
          <tr
            v-for="(row, rowIndex) in rowsArray"
            :key="String(keyOf(row) ?? rowIndex)"
            :style="{ height: `${rowHeight}px` }"
            @click="onRowClick(row, rowIndex, $event)"
            @dblclick="onRowDblclick(row, rowIndex, $event)"
            @contextmenu="onRowContextmenu(row, rowIndex, $event)"
          >
            <td v-for="col in columnsToUse" :key="String(col.key ?? col.dataKey)" :style="tdStyle(col)">
              <component :is="() => renderCell(col, row, rowIndex)" />
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="footerHeight" class="ovtable__footer" :style="{ height: `${footerHeight}px` }">
        <slot name="footer" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, Ref, ref, isRef, h } from 'vue';
import {
  useVTableSelection,
  useVTableProvideColumnRegistry,
  useVTableProvideBuildContext,
  type Column,
} from '@/web/web/composables/useVTable';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/vtable/VTable' });

export type RowEventHandlerParams = {
  rowData: any;
  rowIndex: number;
  rowKey?: string | number;
  event?: Event;
};

const props = withDefaults(
  defineProps<{
    data: any[] | (() => any[]) | any;
    columns?: Column[];
    rowKey?: string;
    rowHeight?: number;
    headerHeight?: number;
    tableHeight?: number;
    footerHeight?: number;
    selectionApi?: ReturnType<typeof useVTableSelection>;
    baseIndex?: number | Ref<number>;
    store?: any;
  }>(),
  { rowKey: 'Id', rowHeight: 40, headerHeight: 48, footerHeight: 0 }
);

const rowsArray = computed<any[]>(() => {
  const src: any = props.data;
  try {
    if (typeof src === 'function') {
      const r = src();
      return Array.isArray(r) ? r : (r ?? []);
    }
    if (isRef(src)) {
      const v = src.value;
      return Array.isArray(v) ? v : (v ?? []);
    }
    return Array.isArray(src) ? src : (src ?? []);
  } catch {
    return [];
  }
});

const baseIndexRef = computed(() => {
  const p = (props.store as any)?.state?.pagination;
  if (p) return (p.currentPage - 1) * p.pageSize + 1;
  const bi = props.baseIndex as any;
  return typeof bi === 'number' ? bi : 1;
});

useVTableProvideBuildContext({
  selectionApi: props.selectionApi,
  getRows: () => rowsArray.value,
  baseIndex: baseIndexRef,
  store: props.store,
});

const { columns: regColumns } = useVTableProvideColumnRegistry();

const columnsToUse = computed<Column[]>(() => {
  return props.columns && props.columns.length > 0 ? props.columns : regColumns.value;
});

const emit = defineEmits<{
  (e: 'row-click', payload: RowEventHandlerParams): void;
  (e: 'row-contextmenu', payload: RowEventHandlerParams): void;
  (e: 'row-dblclick', payload: RowEventHandlerParams): void;
  (e: 'row-mouse-enter', payload: RowEventHandlerParams): void;
  (e: 'row-mouse-leave', payload: RowEventHandlerParams): void;
  (e: 'scroll', ev: any): void;
  (e: 'selection-change', rows: any[]): void;
  (e: 'sort-change', payload: { field: string; direction?: 'asc' | 'desc' }): void;
}>();

const tableRef = ref<HTMLElement | null>(null);

const keyOf = (row: any) =>
  row?.key ??
  row?.__rowKey ??
  row?.[props.rowKey!] ??
  row?.Id ??
  (typeof row === 'object' ? (row as any)?.payload?.Id : undefined);

const selectedItems = computed(() => {
  const set = props.selectionApi?.selected.value ?? new Set<string | number>();
  const src = rowsArray.value || [];
  return src.filter(r => set.has(keyOf(r)));
});

watch(
  () => props.selectionApi?.selected.value,
  () => emit('selection-change', selectedItems.value)
);

function emitSortChange(payload: { field: string; direction?: 'asc' | 'desc' }) {
  emit('sort-change', payload);
}

defineExpose({
  selectedItems,
  emitSortChange,
  scrollTo: (_left?: number, _top?: number) => {},
  scrollToRow: (_rowIndex: number, _align?: string) => {},
});

function thStyle(col: Column) {
  return {
    width: col.width ? `${col.width}px` : undefined,
    minWidth: col.minWidth ? `${col.minWidth}px` : undefined,
    textAlign: col.align || 'left',
  };
}

function tdStyle(col: Column) {
  return thStyle(col);
}

function renderHeader(col: Column) {
  if (typeof col.headerCellRenderer === 'function') {
    return col.headerCellRenderer();
  }
  return h('span', null, col.title ?? '');
}

function renderCell(col: Column, row: any, rowIndex: number) {
  if (typeof col.cellRenderer === 'function') {
    return col.cellRenderer({ rowData: row, rowIndex, column: col });
  }
  const key = String(col.dataKey ?? '');
  const val = key ? row?.[key] : undefined;
  return h('span', null, val == null ? '' : String(val));
}

function onHeaderClick(col: Column) {
  if (!col.sortable) return;
  const field = String(col.dataKey ?? col.key ?? '');
  if (!field) return;
  emit('sort-change', { field, direction: 'asc' });
}

function onRowClick(row: any, rowIndex: number, event: Event) {
  emit('row-click', { rowData: row, rowIndex, rowKey: keyOf(row), event });
}
function onRowDblclick(row: any, rowIndex: number, event: Event) {
  emit('row-dblclick', { rowData: row, rowIndex, rowKey: keyOf(row), event });
}
function onRowContextmenu(row: any, rowIndex: number, event: Event) {
  emit('row-contextmenu', { rowData: row, rowIndex, rowKey: keyOf(row), event });
}
</script>

<style scoped>
.o-vtable {
  display: flex;
  flex-direction: column;
  min-height: 0;
  width: 100%;
}
.ovtable__registrars {
  display: none;
}
.ovtable__scroll {
  overflow: auto;
  flex: 1;
  min-height: 0;
}
.ovtable__table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}
.ovtable__table th,
.ovtable__table td {
  border-bottom: 1px solid var(--choy-color-border, #e5e7eb);
  padding: 0 8px;
  vertical-align: middle;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ovtable__table th {
  position: sticky;
  top: 0;
  background: var(--choy-color-background, #fff);
  z-index: 1;
  font-weight: 600;
  text-align: left;
}
.ovtable__empty {
  padding: 24px;
  text-align: center;
  color: var(--choy-color-muted-foreground, #6b7280);
}
.ovtable__footer {
  border-top: 1px solid var(--choy-color-border, #e5e7eb);
}
</style>
