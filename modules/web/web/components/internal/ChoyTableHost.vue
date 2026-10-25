<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.internal.table-host"
    :data-has-footer="hasFooter ? '' : undefined"
    :data-empty="isEmpty ? '' : undefined"
    :class="
      cn(
        'choy-table-host flex w-full min-h-0 flex-col',
        sizeToContent ? undefined : 'overflow-hidden',
        !sizeToContent && !tableHeight && 'h-full',
        props.class,
      )
    "
    :style="hostRootStyle"
  >
    <!-- Hidden slot used only for ChoyTableColumn registration. -->
    <div class="hidden" aria-hidden="true">
      <slot />
    </div>

    <div
      class="relative min-h-0"
      :class="
        sizeToContent
          ? tableHeight
            ? 'overflow-auto'
            : 'overflow-visible'
          : 'flex-1 overflow-hidden'
      "
    >
      <DataTable
        ref="dataTableRef"
        :class="dataTableClass"
        :columns="columnDefs"
        :data="rowsAsRecords"
        :row-id="resolveRowId"
        :height="dataTableHeight"
        :estimate-size="rowHeight"
        :virtualize="virtualize"
        :enable-row-selection="false"
        :row-selection="selectedRowIds"
        :enable-sorting="true"
        :show-empty="!sizeToContent"
        sorting-mode="server"
        @row-click="onDataTableRowClick"
        @sort-change="onSortChange"
      >
        <!-- Fixed-height hosts: empty lives in the body so it never covers the header. -->
        <template v-if="!sizeToContent" #empty>
          <slot name="empty">
            <div class="p-6 text-center text-muted-foreground">{{ _t('No data') }}</div>
          </slot>
        </template>
      </DataTable>
      <!--
        Content-sized embeds: in-flow empty so the host grows (absolute overlay does not
        expand the parent and gets clipped by the outer overflow-hidden M2M frame).
        Skip when a footer strip is present — Add a line is the empty affordance.
      -->
      <div
        v-if="sizeToContent && isEmpty && !hasFooter"
        class="w-full"
        data-testid="choy-table-host-empty"
      >
        <slot name="empty">
          <div class="p-6 text-center text-muted-foreground">{{ _t('No data') }}</div>
        </slot>
      </div>
    </div>

    <div v-if="footerHeight" class="shrink-0 border-t border-border" :style="{ height: `${footerHeight}px` }">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, ref, isRef, type Ref } from 'vue';
import {
  useTableSelection,
  useTableProvideColumnRegistry,
  useTableProvideBuildContext,
  type Column,
} from '@/web/web/composables/useTable';
import { createTranslate } from '@/web/web/i18n';
import { cn, type ClassValue } from '@/web/web/lib/utils';
import { CHOY_LIST_HEADER_HEIGHT, CHOY_LIST_ROW_HEIGHT } from '@/web/web/composables/listDensity';
import DataTable from './DataTable.vue';
import { columnsToColumnDefs } from './columnAdapter';
import type { DataTableRowId } from './dataTableHelpers';
import type { RowEventHandlerParams } from '@/web/web/components/view/listViewTypes';

const { _t } = createTranslate('web', { scope: 'web/components/internal/ChoyTableHost' });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    data: any[] | (() => any[]) | any;
    columns?: Column[];
    rowKey?: string;
    rowHeight?: number;
    headerHeight?: number;
    tableHeight?: number;
    footerHeight?: number;
    /** When false, skip TanStack virtualizer (embedded M2M/O2M form tables). */
    virtualize?: boolean;
    /**
     * When true, drop DataTable's own border/radius so an outer host frame
     * (e.g. M2M edit chrome) owns the outline without double borders.
     */
    frameless?: boolean;
    selectionApi?: ReturnType<typeof useTableSelection>;
    baseIndex?: number | Ref<number>;
    store?: any;
  }>(),
  {
    rowKey: 'Id',
    rowHeight: CHOY_LIST_ROW_HEIGHT,
    headerHeight: CHOY_LIST_HEADER_HEIGHT,
    footerHeight: 0,
    virtualize: true,
    frameless: false,
  },
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
  const bi = props.baseIndex;
  if (typeof bi === 'number') return bi;
  if (isRef(bi)) {
    const v = Number(bi.value);
    return Number.isFinite(v) ? v : 1;
  }
  return 1;
});

useTableProvideBuildContext({
  selectionApi: props.selectionApi,
  getRows: () => rowsArray.value,
  baseIndex: baseIndexRef,
  store: props.store,
});

const { columns: regColumns } = useTableProvideColumnRegistry();

const columnsToUse = computed<Column[]>(() => {
  return props.columns && props.columns.length > 0 ? props.columns : regColumns.value;
});

const columnDefs = computed(() => columnsToColumnDefs(columnsToUse.value));

/**
 * Non-virtual embedded tables size to their rows (maxHeight caps long lists).
 * Fixed-height + h-full DataTable is what created one-row scrollbars.
 */
const sizeToContent = computed(() => props.virtualize === false);

const hostRootStyle = computed(() => {
  const total = props.tableHeight;
  if (total == null || !Number.isFinite(total)) return undefined;
  if (sizeToContent.value) return { maxHeight: `${total}px` };
  return { height: `${total}px` };
});

const dataTableClass = computed(() =>
  cn(
    !sizeToContent.value && 'h-full',
    props.frameless && 'rounded-none border-0',
    // Content-sized embeds: do not keep a body scrollport that paints a gutter for one row.
    sizeToContent.value && '[&_.choy-data-table__body]:!overflow-y-visible',
  ),
);

const hasFooter = computed(() => (props.footerHeight ?? 0) > 0);
const isEmpty = computed(() => rowsArray.value.length === 0);

/**
 * Fallback body estimate for standalone mounts; flex hosts size via h-full.
 * Embedded non-virtual tables omit a fixed DataTable height so rows define size.
 * When footerHeight is set, tableHeight is the full outer size (header + body + footer).
 */
const bodyHeight = computed(() => {
  const total = props.tableHeight;
  if (total == null || !Number.isFinite(total)) {
    return 280;
  }
  const header = props.headerHeight ?? CHOY_LIST_HEADER_HEIGHT;
  const footer = Math.max(0, props.footerHeight ?? 0);
  const body = Math.max(0, total - header - footer);
  if (props.virtualize === false) return body;
  return Math.max(80, body);
});

/** 0 → DataTable skips fixed root height (prop default 280 would otherwise win over undefined). */
const dataTableHeight = computed(() => (sizeToContent.value ? 0 : bodyHeight.value));

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
  return (rowsArray.value || []).filter(r => set.has(keyOf(r)));
});

/** Mirror host selection into DataTable row state for selected-row chrome (no built-in select column). */
const selectedRowIds = computed<DataTableRowId[] | undefined>(() => {
  if (!props.selectionApi) return undefined;
  return [...(props.selectionApi.selected.value ?? new Set())] as DataTableRowId[];
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
  } as RowEventHandlerParams);
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
/* Footer owns the separator; suppress the last body row border-b to avoid a double rule. */
.choy-table-host[data-has-footer] :deep(.choy-data-table__body [role='row']:last-child) {
  border-bottom-width: 0;
}
/* Empty + footer: header border-b + footer border-t sit flush — keep a single rule. */
.choy-table-host[data-has-footer][data-empty] :deep(.choy-data-table__header) {
  border-bottom-width: 0;
}
</style>
