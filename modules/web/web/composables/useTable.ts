// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { provide, inject, ref, shallowRef, isRef, type Ref, type VNodeChild } from 'vue';

/** Column definition registered by ChoyTableColumn for ChoyTableHost / DataTable. */
export type Column = {
  key?: string | number;
  dataKey?: string;
  title?: string;
  width?: number;
  minWidth?: number;
  align?: 'left' | 'center' | 'right';
  fixed?: 'left' | 'right' | boolean;
  sortable?: boolean;
  headerCellRenderer?: (props?: any) => VNodeChild;
  cellRenderer?: (props: { rowData: any; rowIndex: number; column?: Column }) => VNodeChild;
  [key: string]: any;
};

/**
 * Row-selection modes supported by list / relation tables.
 */
export type SelectionMode = 'multiple' | 'single';

/**
 * Manages row selection state for ChoyTableHost (checkbox columns via ChoyTableColumn).
 */
export function useTableSelection(keyGetter: (row: any) => string | number | undefined | null, mode: SelectionMode = 'multiple') {
  const selected = ref<Set<string | number>>(new Set());
  const modeRef = ref<SelectionMode>(mode);

  const keyOf = (row: any) => keyGetter?.(row);

  function clear() {
    selected.value = new Set();
  }

  function isSelected(row: any) {
    const k = keyOf(row);
    return k != null && selected.value.has(k as any);
  }

  function selectOnly(row: any) {
    const k = keyOf(row);
    selected.value = k != null ? new Set([k as any]) : new Set();
  }

  function toggleRow(row: any, on?: boolean) {
    const k = keyOf(row);
    if (k == null) return;
    if (modeRef.value === 'single') {
      const willOn = on ?? !isSelected(row);
      if (willOn) selectOnly(row);
      else clear();
      return;
    }
    const set = new Set(selected.value);
    const willOn = on ?? !set.has(k as any);
    if (willOn) set.add(k as any);
    else set.delete(k as any);
    selected.value = set;
  }

  function toggleAll(rows: any[], on: boolean) {
    if (modeRef.value === 'single') {
      if (on && rows && rows.length > 0) selectOnly(rows[0]);
      else clear();
      return;
    }
    if (on) {
      const set = new Set<string | number>(selected.value);
      rows?.forEach(r => {
        const k = keyOf(r);
        if (k != null) set.add(k as any);
      });
      selected.value = set;
    } else {
      const set = new Set<string | number>(selected.value);
      rows?.forEach(r => {
        const k = keyOf(r);
        if (k != null) set.delete(k as any);
      });
      selected.value = set;
    }
  }

  function setMode(m: SelectionMode) {
    if (m === modeRef.value) return;
    modeRef.value = m;
    if (m === 'single' && selected.value.size > 1) {
      const first = Array.from(selected.value)[0];
      selected.value = first != null ? new Set([first]) : new Set();
    }
  }

  return { selected, mode: modeRef, isSelected, selectOnly, toggleRow, toggleAll, clear, setMode };
}

/**
 * Column registry shared by ChoyTableColumn under ChoyTableHost.
 */
export type ColumnRegistry = {
  columns: Ref<Column[]>;
  register: (c: Column) => () => void;
};
const TABLE_COLREG_KEY = Symbol('choy-table:col-reg');

/**
 * Provides a column registry for nested table column components.
 */
export function useTableProvideColumnRegistry(): ColumnRegistry {
  const columns = ref<Column[]>([]);
  function register(col: Column) {
    columns.value.push(col);
    return () => {
      const i = columns.value.indexOf(col);
      if (i >= 0) columns.value.splice(i, 1);
    };
  }
  provide(TABLE_COLREG_KEY, { columns, register });
  return { columns, register };
}

/**
 * Injects the current table column registry if available.
 */
export function useTableUseColumnRegistry(): ColumnRegistry | null {
  return inject<ColumnRegistry | null>(TABLE_COLREG_KEY, null);
}

/**
 * Build context shared by ChoyTableColumn (selection, index base, store).
 */
export type TableBuildContext = {
  selectionApi?: ReturnType<typeof useTableSelection>;
  getRows?: () => any[];
  baseIndex?: Ref<number>;
  store?: any;
};
type TableBuildContextInput = {
  selectionApi?: ReturnType<typeof useTableSelection>;
  getRows?: () => any[];
  baseIndex?: number | Ref<number>;
  store?: any;
};
const TABLE_BUILDCTX_KEY = Symbol('choy-table:build-ctx');

/**
 * Provides build context for nested ChoyTableColumn components.
 */
export function useTableProvideBuildContext(ctx: TableBuildContextInput): TableBuildContext {
  const normalized: TableBuildContext = {
    selectionApi: ctx.selectionApi,
    getRows: ctx.getRows,
    baseIndex: isRef(ctx.baseIndex) ? ctx.baseIndex : ref(ctx.baseIndex ?? 1),
    store: ctx.store,
  };
  provide(TABLE_BUILDCTX_KEY, normalized);
  return normalized;
}

/**
 * Injects the current table build context.
 */
export function useTableUseBuildContext(): TableBuildContext {
  return inject<TableBuildContext>(TABLE_BUILDCTX_KEY, {});
}

/**
 * Internal metadata attached to columns for width semantics.
 */
export type TableColumnMeta = {
  widthSpec?: { type: 'percent'; ratio: number } | { type: 'flex'; weight: number } | { type: 'auto' };
};
const TABLE_COL_META_KEY = Symbol('choy-table:col-meta');

/**
 * Stores internal width metadata on a table column.
 */
export function setTableColumnMeta(col: Column, meta: TableColumnMeta) {
  const anyCol = col as unknown as Record<PropertyKey, any>;
  anyCol[TABLE_COL_META_KEY] = { ...(anyCol[TABLE_COL_META_KEY] || {}), ...meta };
}

/**
 * Reads internal width metadata from a table column.
 */
export function getTableColumnMeta(col: Column): TableColumnMeta | undefined {
  return (col as unknown as Record<PropertyKey, any>)[TABLE_COL_META_KEY];
}
