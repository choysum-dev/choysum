// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { provide, inject, ref, shallowRef, isRef, type Ref } from 'vue';
import type { Column } from 'element-plus';

/**
 * Row-selection modes supported by virtual tables.
 */
export type SelectionMode = 'multiple' | 'single';

/**
 * Manages row selection state for virtual tables.
 */
export function useVTableSelection(keyGetter: (row: any) => string | number | undefined | null, mode: SelectionMode = 'multiple') {
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
 * Column registry shared by OVTable column components.
 */
export type ColumnRegistry = {
  columns: Ref<import('element-plus').Column[]>;
  register: (c: import('element-plus').Column) => () => void;
};
const VTABLE_COLREG_KEY = Symbol('ovtable:col-reg');

/**
 * Provides a column registry for nested table column components.
 */
export function useVTableProvideColumnRegistry(): ColumnRegistry {
  const columns = ref<import('element-plus').Column[]>([]);
  function register(col: import('element-plus').Column) {
    columns.value.push(col);
    return () => {
      const i = columns.value.indexOf(col);
      if (i >= 0) columns.value.splice(i, 1);
    };
  }
  provide(VTABLE_COLREG_KEY, { columns, register });
  return { columns, register };
}

/**
 * Injects the current table column registry if available.
 */
export function useVTableUseColumnRegistry(): ColumnRegistry | null {
  return inject<ColumnRegistry | null>(VTABLE_COLREG_KEY, null);
}

/**
 * Build context shared by virtual table columns.
 */
export type VTableBuildContext = {
  selectionApi?: ReturnType<typeof useVTableSelection>;
  getRows?: () => any[];
  baseIndex?: Ref<number>;
  store?: any;
};
type VTableBuildContextInput = {
  selectionApi?: ReturnType<typeof useVTableSelection>;
  getRows?: () => any[];
  baseIndex?: number | Ref<number>;
  store?: any;
};
const VTABLE_BUILDCTX_KEY = Symbol('ovtable:build-ctx');

/**
 * Provides build context for virtual table columns.
 */
export function useVTableProvideBuildContext(ctx: VTableBuildContextInput): VTableBuildContext {
  const normalized: VTableBuildContext = {
    selectionApi: ctx.selectionApi,
    getRows: ctx.getRows,
    baseIndex: isRef(ctx.baseIndex) ? ctx.baseIndex : ref(ctx.baseIndex ?? 1),
    store: ctx.store,
  };
  provide(VTABLE_BUILDCTX_KEY, normalized);
  return normalized;
}

/**
 * Injects the current virtual table build context.
 */
export function useVTableUseBuildContext(): VTableBuildContext {
  return inject<VTableBuildContext>(VTABLE_BUILDCTX_KEY, {});
}

/**
 * Internal metadata attached to columns for width semantics.
 */
export type OVColumnMeta = {
  widthSpec?: { type: 'percent'; ratio: number } | { type: 'flex'; weight: number } | { type: 'auto' };
};
const OV_META_KEY = Symbol('ov:col-meta');

/**
 * Stores internal width metadata on a table column.
 */
export function setOVColumnMeta(col: Column, meta: OVColumnMeta) {
  const anyCol = col as unknown as Record<PropertyKey, any>;
  anyCol[OV_META_KEY] = { ...(anyCol[OV_META_KEY] || {}), ...meta };
}

/**
 * Reads internal width metadata from a table column.
 */
export function getOVColumnMeta(col: Column): OVColumnMeta | undefined {
  return (col as unknown as Record<PropertyKey, any>)[OV_META_KEY];
}
