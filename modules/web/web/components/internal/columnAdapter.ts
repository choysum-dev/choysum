// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type ColumnDef } from '@tanstack/vue-table';
import type { Column } from '@/web/web/composables/useTable';

const DEFAULT_COLUMN_SIZE = 150;

function columnKey(col: Column): string {
  return String(col.key ?? col.dataKey ?? '');
}

/**
 * Stable list column order for grouped mode: selection → handle → index →
 * group label → remaining field columns (matches main OVTable normalizeColumns).
 */
export function orderListTableColumns(columns: readonly Column[]): Column[] {
  if (!columns.length) return [];
  const isSel = (c: Column) => {
    const k = columnKey(c);
    return k === '__selection__' || k === '__select';
  };
  const isHandle = (c: Column) => columnKey(c) === '__handle__' || c.type === 'handle';
  const isIdx = (c: Column) => columnKey(c) === '__index__' || c.type === 'index';
  const isGroup = (c: Column) => columnKey(c) === '__group_label';

  const sel: Column[] = [];
  const handle: Column[] = [];
  const idx: Column[] = [];
  const group: Column[] = [];
  const rest: Column[] = [];
  for (const c of columns) {
    if (isSel(c)) sel.push(c);
    else if (isHandle(c)) handle.push(c);
    else if (isIdx(c)) idx.push(c);
    else if (isGroup(c)) group.push(c);
    else rest.push(c);
  }
  return [...sel, ...handle, ...idx, ...group, ...rest];
}

/**
 * Maps ChoyTableColumn registry entries (legacy Column) to TanStack ColumnDef for DataTable.
 * Cell/header renderers stay on the Column so slot-built VNodes keep working.
 */
export function columnsToColumnDefs<T extends Record<string, unknown>>(
  columns: readonly Column[],
): ColumnDef<T, unknown>[] {
  return orderListTableColumns(columns).map((col, index) => {
    const id = String(col.key ?? col.dataKey ?? `col_${index}`);
    const size =
      typeof col.width === 'number' && col.width > 0 ? col.width : DEFAULT_COLUMN_SIZE;
    const minSize =
      typeof col.minWidth === 'number' && col.minWidth > 0 ? col.minWidth : undefined;

    const def: ColumnDef<T, unknown> = {
      id,
      header: () => {
        if (typeof col.headerCellRenderer === 'function') {
          return col.headerCellRenderer();
        }
        return col.title ?? '';
      },
      cell: (info) => {
        if (typeof col.cellRenderer === 'function') {
          return col.cellRenderer({
            rowData: info.row.original,
            rowIndex: info.row.index,
            column: col,
          });
        }
        const key = String(col.dataKey ?? '');
        const val = key ? (info.row.original as Record<string, unknown>)[key] : undefined;
        return val == null ? '' : String(val);
      },
      size,
      enableSorting: !!col.sortable,
      meta: {
        align: col.align,
        fixed: col.fixed,
        // List host selection column (ChoyTableColumn type="selection").
        selectColumn: id === '__selection__' || id === '__select',
      },
    };
    if (minSize != null) {
      def.minSize = minSize;
    }
    if (col.dataKey != null) {
      (def as { accessorKey?: string }).accessorKey = String(col.dataKey);
    }
    return def;
  });
}
