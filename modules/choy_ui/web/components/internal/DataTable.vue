<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts" generic="T extends Record<string, unknown>">
import { computed, ref, watch } from 'vue';
import {
  FlexRender,
  getCoreRowModel,
  getSortedRowModel,
  useVueTable,
  type ColumnDef,
  type RowSelectionState,
  type SortingState,
} from '@tanstack/vue-table';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { cn, type ClassValue } from '../../lib/utils';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import {
  mapDataTableSelectionKeys,
  nextDataTableSort,
  resolveDataTableRowId,
  type DataTableRowId,
} from './dataTableHelpers';

/**
 * L3 virtual data table (TanStack Table + virtualizer).
 * Not a public Choy* export — consumed by List / Gallery dogfood.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    columns: ColumnDef<T, unknown>[];
    data: T[];
    rowId?: (row: T) => DataTableRowId;
    height?: number;
    estimateSize?: number;
    enableSorting?: boolean;
    enableRowSelection?: boolean;
  }>(),
  {
    height: 280,
    estimateSize: 36,
    enableSorting: true,
    enableRowSelection: true,
  },
);

const emit = defineEmits<{
  'update:rowSelection': [ids: DataTableRowId[]];
  'row-click': [row: T];
}>();

const sorting = ref<SortingState>([]);
const rowSelection = ref<RowSelectionState>({});
/** TanStack keys are always strings; keep originals for emit typing. */
const idRegistry = new Map<string, DataTableRowId>();

const selectColumn = computed<ColumnDef<T, unknown>[]>(() => {
  if (!props.enableRowSelection) {
    return [];
  }
  return [
    {
      id: '__select',
      size: 44,
      enableSorting: false,
      header: () => '',
      cell: () => '',
    },
  ];
});

const table = useVueTable({
  get data() {
    return props.data;
  },
  get columns() {
    return [...selectColumn.value, ...props.columns];
  },
  state: {
    get sorting() {
      return sorting.value;
    },
    get rowSelection() {
      return rowSelection.value;
    },
  },
  get enableRowSelection() {
    return props.enableRowSelection;
  },
  get enableSorting() {
    return props.enableSorting;
  },
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  onSortingChange: (updater) => {
    sorting.value = typeof updater === 'function' ? updater(sorting.value) : updater;
  },
  onRowSelectionChange: (updater) => {
    rowSelection.value = typeof updater === 'function' ? updater(rowSelection.value) : updater;
  },
  getRowId: (row) => {
    const original = resolveDataTableRowId(row, props.rowId);
    const key = String(original);
    idRegistry.set(key, original);
    return key;
  },
});

const parentRef = ref<HTMLElement | null>(null);
const headerRef = ref<HTMLElement | null>(null);
const rows = computed(() => table.getRowModel().rows);

const virtualizer = useVirtualizer({
  get count() {
    return rows.value.length;
  },
  getScrollElement: () => parentRef.value as Element | null,
  estimateSize: () => props.estimateSize,
  overscan: 8,
});

const virtualRows = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());
const gridTemplate = computed(() =>
  table
    .getVisibleLeafColumns()
    .map((col) => `${col.getSize()}px`)
    .join(' '),
);
const tableMinWidth = computed(() =>
  table.getVisibleLeafColumns().reduce((sum, col) => sum + col.getSize(), 0),
);

watch(
  rowSelection,
  (state) => {
    const keys = Object.keys(state).filter((key) => state[key]);
    emit('update:rowSelection', mapDataTableSelectionKeys(keys, idRegistry));
  },
  { deep: true },
);

function onHeaderClick(columnId: string, canSort: boolean): void {
  if (!canSort || !props.enableSorting || columnId === '__select') {
    return;
  }
  const current =
    sorting.value[0] != null
      ? { id: sorting.value[0].id, desc: !!sorting.value[0].desc }
      : null;
  const next = nextDataTableSort(current, columnId);
  sorting.value = next ? [{ id: next.id, desc: next.desc }] : [];
}

const allSelected = computed(() => {
  if (table.getIsAllPageRowsSelected()) {
    return true;
  }
  if (table.getIsSomePageRowsSelected()) {
    return 'indeterminate' as const;
  }
  return false;
});

let syncingScroll = false;
function onBodyScroll(): void {
  if (syncingScroll || !parentRef.value || !headerRef.value) {
    return;
  }
  syncingScroll = true;
  headerRef.value.scrollLeft = parentRef.value.scrollLeft;
  syncingScroll = false;
}

function onHeaderScroll(): void {
  if (syncingScroll || !parentRef.value || !headerRef.value) {
    return;
  }
  syncingScroll = true;
  parentRef.value.scrollLeft = headerRef.value.scrollLeft;
  syncingScroll = false;
}
</script>

<template>
  <div
    data-anchor="choy.internal.data-table"
    :class="cn('choy-data-table overflow-hidden rounded-md border border-border bg-background', props.class)"
  >
    <div
      ref="headerRef"
      class="choy-data-table__header overflow-x-auto overflow-y-hidden border-b border-border bg-muted/40 text-xs font-medium text-foreground/80"
      @scroll.passive="onHeaderScroll"
    >
      <div
        class="grid"
        :style="{ minWidth: `${tableMinWidth}px`, gridTemplateColumns: gridTemplate }"
      >
        <div
          v-for="header in table.getHeaderGroups()[0]?.headers ?? []"
          :key="header.id"
          class="flex items-center gap-1 px-2 py-2"
        >
          <Checkbox
            v-if="header.column.id === '__select'"
            :model-value="allSelected"
            aria-label="Select all"
            @update:model-value="(v: boolean | 'indeterminate') => table.toggleAllPageRowsSelected(v === true)"
          />
          <button
            v-else
            type="button"
            class="flex flex-1 items-center gap-1 text-left hover:text-foreground"
            :class="{ 'cursor-default': !header.column.getCanSort() }"
            @click="onHeaderClick(header.column.id, header.column.getCanSort())"
          >
            <FlexRender :render="header.column.columnDef.header" :props="header.getContext()" />
            <span v-if="header.column.getIsSorted()" class="text-[10px] text-foreground/50">
              {{ header.column.getIsSorted() === 'desc' ? '↓' : '↑' }}
            </span>
          </button>
        </div>
      </div>
    </div>

    <div
      ref="parentRef"
      class="choy-data-table__body relative overflow-auto"
      :style="{ height: `${height}px` }"
      @scroll.passive="onBodyScroll"
    >
      <div
        :style="{
          height: `${totalSize}px`,
          position: 'relative',
          minWidth: `${tableMinWidth}px`,
          width: '100%',
        }"
      >
        <div
          v-for="virtualRow in virtualRows"
          :key="String(rows[virtualRow.index]?.id ?? virtualRow.key)"
          class="absolute left-0 grid w-full border-b border-border/60 text-sm hover:bg-muted/30"
          :style="{
            transform: `translateY(${virtualRow.start}px)`,
            height: `${virtualRow.size}px`,
            gridTemplateColumns: gridTemplate,
          }"
          @click="rows[virtualRow.index] && emit('row-click', rows[virtualRow.index].original)"
        >
          <div
            v-for="cell in rows[virtualRow.index]?.getVisibleCells() ?? []"
            :key="cell.id"
            class="flex items-center truncate px-2"
          >
            <Checkbox
              v-if="cell.column.id === '__select'"
              :model-value="cell.row.getIsSelected()"
              aria-label="Select row"
              @click.stop
              @update:model-value="(v: boolean | 'indeterminate') => cell.row.toggleSelected(v === true)"
            />
            <FlexRender
              v-else
              :render="cell.column.columnDef.cell"
              :props="cell.getContext()"
            />
          </div>
        </div>
      </div>
      <div
        v-if="!rows.length"
        class="flex h-full items-center justify-center text-sm text-foreground/50"
      >
        No data
      </div>
    </div>
  </div>
</template>
