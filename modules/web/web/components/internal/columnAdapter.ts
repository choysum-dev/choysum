// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type ColumnDef } from '@tanstack/vue-table';
import type { Column } from '@/web/web/composables/useVTable';

const DEFAULT_COLUMN_SIZE = 150;

/**
 * Maps ChoyVColumn registry entries (legacy Column) to TanStack ColumnDef for DataTable.
 * Cell/header renderers stay on the Column so slot-built VNodes keep working.
 */
export function columnsToColumnDefs<T extends Record<string, unknown>>(
  columns: readonly Column[],
): ColumnDef<T, unknown>[] {
  return columns.map((col, index) => {
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
