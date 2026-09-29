<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-table-host" :style="{ height: tableHeight ? `${tableHeight}px` : undefined }">
    <!-- Hidden slot used only for ChoyVColumn registration. -->
    <div class="choy-table-host__registrars" aria-hidden="true">
      <slot />
    </div>

    <div class="choy-table-host__body" :style="{ height: `${bodyHeight}px` }">
      <DataTable
        ref="dataTableRef"
        class="choy-table-host__table"
        :columns="columnDefs"
        :data="rowsAsRecords"
        :row-id="resolveRowId"
        :height="bodyHeight"
        :estimate-size="rowHeight"
        :enable-row-selection="false"
        :enable-sorting="true"
        :show-empty="false"
        sorting-mode="server"
        @row-click="onDataTableRowClick"
        @sort-change="onSortChange"
      />
      <div v-if="!rowsArray.length" class="choy-table-host__empty-wrap">
        <slot name="empty">
          <div class="choy-table-host__empty">{{ _t('No data') }}</div>
        </slot>
      </div>
    </div>

    <div v-if="footerHeight" class="choy-table-host__footer" :style="{ height: `${footerHeight}px` }">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, ref, isRef, type Ref } from 'vue';
import {
  useVTableSelection,
  useVTableProvideColumnRegistry,
  useVTableProvideBuildContext,
  type Column,
} from '@/web/web/composables/useVTable';
import { createTranslate } from '@/web/web/i18n';
import DataTable from './DataTable.vue';
import { columnsToColumnDefs } from './columnAdapter';
import type { DataTableRowId } from './dataTableHelpers';
import type { RowEventHandlerParams } from '@/web/web/components/view/listViewTypes';

const { _t } = createTranslate('web', { scope: 'web/components/internal/ChoyTableHost' });

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
  { rowKey: 'Id', rowHeight: 40, headerHeight: 48, footerHeight: 0 },
);

const emit = defineEmits<{
  (e: 'row-click', payload: RowEventHandlerParams): void;
  (e: 'row-contextmenu', payload: RowEventHandlerParams): void;
  (e: 'row-dblclick', payload: RowEventHandlerParams): void;
  (e: 'selection-change', rows: any[]): void;
  (e: 'sort-change', payload: { field: string; direction?: 'asc' | 'desc' }): void;
}>();

const dataTableRef = ref<{ scrollToRow?: (index: number, align?: string) => void } | null>(null);

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

/** DataTable requires Record rows; keep originals as object identity. */
const rowsAsRecords = computed(() => rowsArray.value as Record<string, unknown>[]);

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

const columnDefs = computed(() => columnsToColumnDefs(columnsToUse.value));

const bodyHeight = computed(() => {
  const total = props.tableHeight;
  if (total == null || !Number.isFinite(total)) {
    return 280;
  }
  const header = props.headerHeight ?? 48;
  return Math.max(80, total - header);
});

const syntheticKeys = new WeakMap<object, string>();
let syntheticSeq = 0;

const keyOf = (row: any): DataTableRowId => {
  const raw =
    row?.key ??
    row?.__rowKey ??
    row?.[props.rowKey!] ??
    row?.Id ??
    (typeof row === 'object' ? (row as any)?.payload?.Id : undefined);
  if (raw != null && raw !== '') {
    return raw as DataTableRowId;
  }
  // Stable synthetic id so rows without Id still virtualize and select.
  if (row && typeof row === 'object') {
    let key = syntheticKeys.get(row);
    if (!key) {
      key = `__row_${++syntheticSeq}`;
      syntheticKeys.set(row, key);
    }
    return key;
  }
  return `__row_${++syntheticSeq}`;
};

function resolveRowId(row: Record<string, unknown>): DataTableRowId {
  return keyOf(row);
}

const selectedItems = computed(() => {
  const set = props.selectionApi?.selected.value ?? new Set<string | number>();
  return (rowsArray.value || []).filter((r) => set.has(keyOf(r)));
});

watch(
  () => props.selectionApi?.selected.value,
  () => emit('selection-change', selectedItems.value),
);

function onDataTableRowClick(row: Record<string, unknown>): void {
  const rowIndex = rowsArray.value.indexOf(row);
  emit('row-click', {
    rowData: row,
    rowIndex: rowIndex >= 0 ? rowIndex : 0,
    rowKey: keyOf(row),
  });
}

function onSortChange(payload: { field: string; direction?: 'asc' | 'desc' }): void {
  emit('sort-change', payload);
}

function emitSortChange(payload: { field: string; direction?: 'asc' | 'desc' }) {
  emit('sort-change', payload);
}

defineExpose({
  selectedItems,
  emitSortChange,
  scrollTo: (_left?: number, _top?: number) => {},
  scrollToRow: (rowIndex: number, align?: string) => {
    dataTableRef.value?.scrollToRow?.(rowIndex, align as 'start' | 'center' | 'end' | 'auto');
  },
});
</script>

<style scoped>
.choy-table-host {
  display: flex;
  flex-direction: column;
  min-height: 0;
  width: 100%;
}
.choy-table-host__registrars {
  display: none;
}
.choy-table-host__body {
  position: relative;
  flex: 1;
  min-height: 0;
}
.choy-table-host__table {
  height: 100%;
}
.choy-table-host__empty-wrap {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
}
.choy-table-host__empty {
  padding: 24px;
  text-align: center;
  color: var(--choy-color-muted-foreground, #6b7280);
}
.choy-table-host__footer {
  border-top: 1px solid var(--choy-color-border, #e5e7eb);
}
</style>
