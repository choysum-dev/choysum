<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    role="grid"
    data-anchor="choy.internal.data-table"
    :aria-multiselectable="enableRowSelection ? 'true' : undefined"
    :aria-rowcount="rows.length ? rows.length + 1 : 2"
    :aria-colcount="table.getVisibleLeafColumns().length"
    :class="cn('choy-data-table overflow-hidden rounded-md border border-border bg-background', props.class)"
  >
    <div
      ref="headerRef"
      role="rowgroup"
      class="choy-data-table__header overflow-x-auto overflow-y-hidden border-b border-border bg-muted/20 text-xs font-medium text-muted-foreground"
      @scroll.passive="onHeaderScroll"
    >
      <div
        role="row"
        aria-rowindex="1"
        class="grid"
        :style="{ minWidth: `${tableMinWidth}px`, gridTemplateColumns: gridTemplate }"
      >
        <div
          v-for="header in (table.getHeaderGroups().slice(-1)[0]?.headers ?? [])"
          :key="header.id"
          role="columnheader"
          class="flex items-center gap-1 px-2 py-1.5"
          :aria-sort="
            header.column.id === '__select' || !header.column.getCanSort()
              ? undefined
              : header.column.getIsSorted() === 'asc'
                ? 'ascending'
                : header.column.getIsSorted() === 'desc'
                  ? 'descending'
                  : 'none'
          "
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
            class="flex flex-1 items-center gap-1 border-0 bg-transparent p-0 text-left font-medium text-muted-foreground hover:text-foreground"
            :class="{ 'cursor-default': !header.column.getCanSort(), 'cursor-pointer': header.column.getCanSort() }"
            :disabled="!header.column.getCanSort()"
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
      role="rowgroup"
      class="choy-data-table__body relative overflow-auto"
      :style="{ height: `${height}px` }"
      @scroll.passive="onBodyScroll"
    >
      <div
        role="none"
        :style="
          isVirtualized
            ? {
                height: `${totalSize}px`,
                position: 'relative',
                minWidth: `${tableMinWidth}px`,
                width: '100%',
              }
            : {
                minWidth: `${tableMinWidth}px`,
                width: '100%',
              }
        "
      >
        <div
          v-for="item in bodyRows"
          :key="String(item.key)"
          :ref="isVirtualized ? measureRowElement : undefined"
          :data-index="item.index"
          role="row"
          :aria-rowindex="item.index + 2"
          :aria-selected="item.row?.getIsSelected() ?? false"
          tabindex="0"
          class="grid w-full border-b border-border/60 text-sm hover:bg-muted/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          :class="item.start != null ? 'absolute left-0' : undefined"
          :style="{
            gridTemplateColumns: gridTemplate,
            ...(item.start != null
              ? { transform: `translateY(${item.start}px)` }
              : undefined),
          }"
          @click="onRowClick($event, item.row)"
          @keydown="onRowKeydown($event, item.row)"
        >
          <div
            v-for="cell in item.row?.getVisibleCells() ?? []"
            :key="cell.id"
            role="gridcell"
            class="flex items-center truncate px-2"
          >
            <Checkbox
              v-if="cell.column.id === '__select'"
              :model-value="cell.row.getIsSelected()"
              :aria-label="`Select row ${item.index + 1}`"
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
        v-if="showEmpty && !rows.length"
        role="row"
        aria-rowindex="2"
        class="flex h-full items-center justify-center text-sm text-foreground/50"
      >
        <div role="gridcell">
          <slot name="empty">No data</slot>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts" generic="T extends Record<string, unknown>">
import { computed, ref, watch } from 'vue';
import {
  FlexRender,
  getCoreRowModel,
  getSortedRowModel,
  useVueTable,
  type ColumnDef,
  type RowSelectionState,
  type SortingState
} from '@tanstack/vue-table';
import { useVirtualizer } from '@tanstack/vue-virtual';
import { cn, type ClassValue } from '../../lib/utils';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';
import {
  applyDataTableParentScroll,
  applyDataTableScrollToRow,
  compareDataTableValues,
  dataTableIsVirtualized,
  dataTableMaybeMeasure,
  dataTableMeasureRow,
  dataTableScrollElement,
  dataTableSelectionIdsEqual,
  dataTableVirtualizerCount,
  decodeDataTableRowKey,
  encodeDataTableRowKey,
  mapDataTableBodyRows,
  mapDataTableSelectionKeys,
  mergeDataTableControlledSelection,
  nextDataTableSort,
  nextServerDataTableSort,
  normalizeDataTableRowId,
  pruneDataTableSelection,
  resolveDataTableEstimateSize,
  resolveDataTableRowId,
  type DataTableRowId
} from './dataTableHelpers';

/**
 * L3 virtual data table (TanStack Table + virtualizer).
 * Not a public Choy* export — consumed by List and related kit hosts.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    columns: ColumnDef<T, unknown>[];
    data: T[];
    rowId?: (row: T) => DataTableRowId;
    /** Controlled selection ids (pairs with update:rowSelection / v-model:rowSelection). */
    rowSelection?: DataTableRowId[] | null;
    height?: number;
    estimateSize?: number;
    /**
     * When false, render every row in normal flow and do not attach a
     * virtualizer ResizeObserver (embedded M2M/O2M tables in overflow-hidden form chrome).
     */
    virtualize?: boolean;
    enableSorting?: boolean;
    /**
     * client: reorder rows in the table.
     * server: header clicks emit sort-change; data order is host-owned.
     */
    sortingMode?: 'client' | 'server';
    enableRowSelection?: boolean;
    /** When false, host renders its own empty state. */
    showEmpty?: boolean;
  }>(),
  {
    height: 280,
    estimateSize: 32,
    virtualize: true,
    enableSorting: true,
    sortingMode: 'client',
    enableRowSelection: true,
    showEmpty: true,
  },
);

const emit = defineEmits<{
  'update:rowSelection': [ids: DataTableRowId[]];
  'row-click': [row: T];
  'sort-change': [payload: { field: string; direction?: 'asc' | 'desc' }];
}>();

const sorting = ref<SortingState>([]);
const rowSelection = ref<RowSelectionState>({});
/** TanStack keys are always strings; keep originals for emit typing. */
const idRegistry = new Map<string, DataTableRowId>();

function isControlledSelection(): boolean {
  return props.rowSelection !== undefined;
}

const presentKeys = computed(() => {
  const keys = new Set<string>();
  props.data.forEach((row, index) => {
    let key: string;
    try {
      key = encodeDataTableRowKey(resolveDataTableRowId(row, props.rowId));
    } catch (error) {
      const reason = error instanceof Error ? error.message : String(error);
      // Name the offending row so hosts can fix `rowId` instead of debugging a blank table.
      throw new Error(`DataTable: row ${index} has no usable Id/id (${reason})`);
    }
    if (keys.has(key)) {
      throw new Error(`DataTable: duplicate row id ${key} at index ${index}`);
    }
    keys.add(key);
  });
  return keys;
});

function presentKeysFromData(): Set<string> {
  return presentKeys.value;
}

function registerSelectionKey(key: string, id?: DataTableRowId): void {
  if (!idRegistry.has(key)) {
    idRegistry.set(key, id ?? decodeDataTableRowKey(key));
  }
}

// Mirror an externally controlled selection without re-emitting identical state.
// Only on-page ids enter TanStack state; off-page ids stay parent-owned.
watch(
  [() => props.rowSelection, presentKeys],
  ([ids]) => {
    if (ids === undefined) {
      return;
    }
    // Nullable v-model hosts may pass null; treat as an empty controlled selection.
    const list = ids ?? [];
    const present = presentKeysFromData();
    const next: RowSelectionState = {};
    for (const raw of list) {
      const id = normalizeDataTableRowId(raw);
      if (id === null) {
        continue;
      }
      const key = encodeDataTableRowKey(id);
      if (!present.has(key)) {
        continue;
      }
      idRegistry.set(key, id);
      next[key] = true;
    }
    const currentKeys = Object.keys(rowSelection.value).filter((key) => rowSelection.value[key]);
    const nextKeys = Object.keys(next);
    if (
      currentKeys.length === nextKeys.length &&
      nextKeys.every((key) => rowSelection.value[key] === true)
    ) {
      return;
    }
    rowSelection.value = next;
  },
  { immediate: true, deep: true },
);

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

const resolvedColumns = computed<ColumnDef<T, unknown>[]>(() => [
  ...selectColumn.value,
  ...props.columns,
]);

const defaultColumn = {
  sortingFn: (
    rowA: { getValue: (columnId: string) => unknown },
    rowB: { getValue: (columnId: string) => unknown },
    columnId: string,
  ) => compareDataTableValues(rowA.getValue(columnId), rowB.getValue(columnId)),
};

const table = useVueTable({
  get data() {
    return props.data;
  },
  get columns() {
    return resolvedColumns.value;
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
  get manualSorting() {
    return props.sortingMode === 'server';
  },
  defaultColumn,
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  onSortingChange: (updater) => {
    sorting.value = typeof updater === 'function' ? updater(sorting.value) : updater;
  },
  onRowSelectionChange: (updater) => {
    const proposed = typeof updater === 'function' ? updater(rowSelection.value) : updater;
    for (const key of Object.keys(proposed)) {
      if (proposed[key]) {
        registerSelectionKey(key);
      }
    }
    const visibleIds = mapDataTableSelectionKeys(
      Object.keys(proposed).filter((key) => proposed[key]),
      idRegistry,
    );
    if (isControlledSelection()) {
      const present = presentKeysFromData();
      const controlledIds = (props.rowSelection ?? [])
        .map(normalizeDataTableRowId)
        .filter((id): id is DataTableRowId => id !== null);
      const nextIds = mergeDataTableControlledSelection(controlledIds, visibleIds, present);
      if (!dataTableSelectionIdsEqual(controlledIds, nextIds)) {
        emit('update:rowSelection', nextIds);
      }
      // Prop remains authoritative; wait for the parent to accept the update.
      return;
    }
    rowSelection.value = proposed;
  },
  getRowId: (row) => {
    const original = resolveDataTableRowId(row, props.rowId);
    const key = encodeDataTableRowKey(original);
    idRegistry.set(key, original);
    return key;
  },
});

const parentRef = ref<HTMLElement | null>(null);
const headerRef = ref<HTMLElement | null>(null);
const rows = computed(() => table.getRowModel().rows);

// Data swaps can drop rows; prune only when selection is uncontrolled.
watch(rows, (current) => {
  const present = new Set(current.map((row) => row.id));
  for (const key of [...idRegistry.keys()]) {
    if (!present.has(key)) {
      idRegistry.delete(key);
    }
  }
  if (isControlledSelection()) {
    return;
  }
  const pruned = pruneDataTableSelection(rowSelection.value, present);
  if (pruned) {
    rowSelection.value = pruned;
  }
});

const isVirtualized = computed(() =>
  dataTableIsVirtualized(props.virtualize, rows.value.length),
);

const virtualizer = useVirtualizer({
  get count() {
    return dataTableVirtualizerCount(isVirtualized.value, rows.value.length);
  },
  // Skip the observer when virtualization is off or the row set is empty.
  getScrollElement: () =>
    dataTableScrollElement(isVirtualized.value, parentRef.value as Element | null),
  estimateSize: () => resolveDataTableEstimateSize(props.estimateSize, 32),
  // Keep measurements attached to the logical row so sorting does not reuse stale heights.
  getItemKey: (index) => rows.value[index]?.id ?? index,
  overscan: 8,
});

// Cached heights win over `estimateSize`; drop them when the estimate changes.
watch(
  () => props.estimateSize,
  () => {
    dataTableMaybeMeasure(isVirtualized.value, virtualizer.value);
  },
);

// Data swaps (paging) must reset offset/heights, but a host that rebuilds an equal
// array must not jump the user back to the top.
watch(
  () => [...presentKeys.value].join('\0'),
  () => {
    if (parentRef.value) {
      parentRef.value.scrollTop = 0;
    }
    dataTableMaybeMeasure(isVirtualized.value, virtualizer.value);
  },
);

// Swapping columns (e.g. another entity) can leave a sort id with no matching column.
watch(
  () => props.columns,
  () => {
    if (sorting.value.some((sort) => !table.getColumn(sort.id))) {
      sorting.value = [];
    }
  },
);

const virtualRows = computed(() => virtualizer.value.getVirtualItems());
const totalSize = computed(() => virtualizer.value.getTotalSize());

const bodyRows = computed(() =>
  mapDataTableBodyRows({
    virtualized: isVirtualized.value,
    rows: rows.value,
    virtualItems: virtualRows.value,
  }),
);

function measureRowElement(el: Element | null): void {
  dataTableMeasureRow(el, isVirtualized.value, virtualizer.value);
}
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
    if (isControlledSelection()) {
      return;
    }
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
  if (props.sortingMode === 'server') {
    // Same cycle as client sort (none → asc → desc → none); host owns row order.
    const next = nextServerDataTableSort(current, columnId);
    sorting.value = next.sorting;
    emit('sort-change', { field: next.field, direction: next.direction });
    return;
  }
  const next = nextDataTableSort(current, columnId);
  sorting.value = next ? [{ id: next.id, desc: next.desc }] : [];
}

function scrollToRow(index: number, align?: 'start' | 'center' | 'end' | 'auto'): void {
  applyDataTableScrollToRow({
    index,
    align,
    virtualized: isVirtualized.value,
    parent: parentRef.value,
    estimateSize: props.estimateSize,
    virtualizer: virtualizer.value,
    setScrollTop: (top) => {
      applyDataTableParentScroll(parentRef.value, top);
    },
  });
}

defineExpose({ scrollToRow });

const allSelected = computed(() => {
  if (table.getIsAllPageRowsSelected()) {
    return true;
  }
  if (table.getIsSomePageRowsSelected()) {
    return 'indeterminate' as const;
  }
  return false;
});

function onBodyScroll(): void {
  const parent = parentRef.value;
  const header = headerRef.value;
  if (!parent || !header || header.scrollLeft === parent.scrollLeft) {
    return;
  }
  header.scrollLeft = parent.scrollLeft;
}

function onHeaderScroll(): void {
  const parent = parentRef.value;
  const header = headerRef.value;
  if (!parent || !header || parent.scrollLeft === header.scrollLeft) {
    return;
  }
  parent.scrollLeft = header.scrollLeft;
}

const interactiveRowClickSelector =
  'a,button,input,textarea,select,label,[role="button"],[role="checkbox"],[role="switch"],[role="menuitem"],[role="menuitemcheckbox"],[role="link"],[role="option"],[role="combobox"],[role="slider"],[contenteditable="true"],[data-no-row-click]';

function isInteractiveRowClickTarget(target: EventTarget | null): boolean {
  return target instanceof Element && !!target.closest(interactiveRowClickSelector);
}

/** Skip row-click when the event originated from an interactive cell control. */
function onRowClick(event: MouseEvent, row: (typeof rows.value)[number] | undefined): void {
  if (!row || isInteractiveRowClickTarget(event.target)) {
    return;
  }
  emit('row-click', row.original);
}

function onRowKeydown(event: KeyboardEvent, row: (typeof rows.value)[number] | undefined): void {
  if (!row || isInteractiveRowClickTarget(event.target)) {
    return;
  }
  if (event.key === ' ') {
    if (!props.enableRowSelection || !row.getCanSelect()) {
      return;
    }
    // ARIA grids toggle the row selection with Space; Enter activates the row.
    event.preventDefault();
    row.toggleSelected();
    return;
  }
  if (event.key !== 'Enter') {
    return;
  }
  event.preventDefault();
  emit('row-click', row.original);
}
</script>
