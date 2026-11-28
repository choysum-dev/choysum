// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { type ColumnDef } from '@tanstack/vue-table';
import {
  getTableColumnMeta,
  type Column,
  type TableColumnMeta,
} from '@/web/web/composables/useTable';
import {
  defaultListColumnMaxWidth,
  estimateListHeaderMinWidth,
  estimateListIndexColumnWidth,
  LIST_INDEX_COLUMN_MIN_WIDTH,
  listColumnIsHard,
  listColumnPrefersStretch,
  maxListIndexDisplayText,
} from '@/web/web/components/field/listColumnSizeDefaults';
import {
  LIST_SOFT_SAMPLE_ROW_LIMIT,
  sampleSoftColumnMinWidths,
  type ListTextMeasureFn,
} from '@/web/web/components/field/listColumnTextMeasure';
import {
  applyUserColumnWidthOverride,
  clampListColumnWidthPx,
} from '@/web/web/composables/listColumnWidthPrefs';

/** Fallback track size when a column has no explicit width (20px scale). */
export const FALLBACK_COLUMN_SIZE = 140;

/**
 * Legacy host flag. Soft/hard sizing is shared by List and embeds; promotion to
 * unbounded `1fr` is no longer applied. Kept so callers need not change props yet.
 */
export type ColumnFillMode = 'none' | 'embed-auto';

export type ColumnsToColumnDefsOptions = {
  columnFillMode?: ColumnFillMode;
  /** First-screen rows for soft-column content min raise (List / O2M). */
  sampleRows?: readonly Record<string, unknown>[];
  /** Injectable text measure (tests); browser uses canvas by default. */
  textMeasure?: ListTextMeasureFn | null;
  /** Cap how many sample rows are scanned (default 20). */
  sampleRowLimit?: number;
  /**
   * User-dragged fixed widths (localStorage). Applied after type/sample defaults;
   * skipped when the author set an explicit width / widthSpec.
   */
  userWidths?: Readonly<Record<string, number>>;
  /** 1-based first row number on the current page (pagination offset + 1). */
  baseIndex?: number;
  /**
   * Precomputed max index display text for the current page. When omitted,
   * derived from sampleRows + baseIndex / groupIndex.
   */
  indexDisplayMaxText?: string;
};

export type ResolveColumnSizeOptions = {
  textMeasure?: ListTextMeasureFn | null;
  /** Soft sample raise for this column key, when present. */
  softSampleMin?: number;
  /** User-dragged fixed width for this column, when present. */
  userWidth?: number;
  /** Max row-number text on the current page (index chrome digit budget). */
  indexDisplayMaxText?: string;
};

function isIndexListColumn(col: Column): boolean {
  return columnKey(col) === '__index__' || col.type === 'index';
}

/**
 * Widest index label on the current page — mirrors ChoyTableColumn index cell:
 * `groupIndex` when set, else `baseIndex + rowIndex`. Uses the full `sampleRows`
 * array (soft `sampleRowLimit` does not trim this path).
 */
export function resolveIndexDisplayMaxText(
  options?: Pick<ColumnsToColumnDefsOptions, 'sampleRows' | 'baseIndex' | 'indexDisplayMaxText'>,
): string {
  const pre = String(options?.indexDisplayMaxText ?? '').trim();
  if (pre) return pre;
  const rows = options?.sampleRows;
  const base =
    typeof options?.baseIndex === 'number' &&
    Number.isFinite(options.baseIndex) &&
    options.baseIndex > 0
      ? Math.floor(options.baseIndex)
      : 1;
  if (!rows || rows.length === 0) {
    return maxListIndexDisplayText({ baseIndex: base, rowCount: 0 });
  }
  let max = 0;
  for (let i = 0; i < rows.length; i++) {
    const gi = (rows[i] as { groupIndex?: unknown } | undefined)?.groupIndex;
    const n =
      typeof gi === 'number' && Number.isFinite(gi) ? gi : base + i;
    if (n > max) max = n;
  }
  return max > 0 ? String(Math.floor(max)) : '';
}

function columnKey(col: Column): string {
  return String(col.key ?? col.dataKey ?? '');
}

function columnPropKey(col: Column): string {
  return String(col.dataKey ?? col.key ?? '');
}

/** Selection / handle / index / group chrome — never soft-stretch. */
export function isSystemListColumn(col: Column): boolean {
  const k = columnKey(col);
  if (
    k === '__selection__' ||
    k === '__select' ||
    k === '__handle__' ||
    k === '__index__' ||
    k === '__group_label'
  ) {
    return true;
  }
  const t = col.type;
  return t === 'selection' || t === 'handle' || t === 'index';
}

function hasAuthorWidthSpec(spec: TableColumnMeta['widthSpec'] | undefined): boolean {
  return (
    spec?.type === 'flex' ||
    spec?.type === 'auto' ||
    spec?.type === 'percent' ||
    spec?.type === 'capped'
  );
}

/**
 * Resolved pixel size for a registry column.
 * Explicit width wins as the base; minWidth alone (type defaults) is the track size;
 * only when neither is set do we use FALLBACK. When both exist, size = max(width, minWidth).
 * Hard columns may raise size via header font/char budget; soft columns may raise via sample.
 */
export function resolveColumnSizePx(
  col: Column,
  opts?: ResolveColumnSizeOptions,
): {
  size: number;
  minSize: number | undefined;
} {
  const explicitWidth =
    typeof col.width === 'number' && col.width > 0 ? col.width : undefined;
  let minSize =
    typeof col.minWidth === 'number' && col.minWidth > 0 ? col.minWidth : undefined;
  let size: number;
  if (explicitWidth != null && minSize != null) {
    size = Math.max(explicitWidth, minSize);
  } else if (explicitWidth != null) {
    size = explicitWidth;
  } else if (minSize != null) {
    size = minSize;
  } else {
    size = FALLBACK_COLUMN_SIZE;
  }

  // Index chrome: digit budget may raise above the floor; never soft/user prefs.
  // Author pixel width raises the floor; digit budget may still grow above it.
  if (isIndexListColumn(col)) {
    const floor =
      explicitWidth != null
        ? Math.max(LIST_INDEX_COLUMN_MIN_WIDTH, explicitWidth)
        : LIST_INDEX_COLUMN_MIN_WIDTH;
    const raised = estimateListIndexColumnWidth(opts?.indexDisplayMaxText, {
      measure: opts?.textMeasure,
    });
    size = Math.max(floor, raised);
    minSize = size;
    return { size, minSize };
  }

  // Header / sample raise only when width was not an author-fixed pixel width on chrome/system.
  if (!isSystemListColumn(col) && explicitWidth == null) {
    const fieldType = getTableColumnMeta(col)?.fieldType;
    if (listColumnIsHard(fieldType)) {
      const raised = estimateListHeaderMinWidth(col.title, size, {
        measure: opts?.textMeasure,
      });
      if (raised > size) {
        size = raised;
        minSize = minSize != null ? Math.max(minSize, raised) : raised;
      }
    } else if (
      listColumnPrefersStretch(fieldType) &&
      typeof opts?.softSampleMin === 'number' &&
      opts.softSampleMin > size
    ) {
      size = opts.softSampleMin;
      minSize = minSize != null ? Math.max(minSize, opts.softSampleMin) : opts.softSampleMin;
    }

    // User drag override (fixed px) beats type/sample, loses to author explicit width.
    const authorSpec = getTableColumnMeta(col)?.widthSpec;
    if (!hasAuthorWidthSpec(authorSpec)) {
      const next = applyUserColumnWidthOverride(size, opts?.userWidth);
      if (next !== size) {
        size = next;
        minSize = next;
      }
    }
  }
  return { size, minSize };
}

/** Whether the column may show a header resize handle. */
export function listColumnIsResizable(col: Column): boolean {
  if (isSystemListColumn(col)) return false;
  // Explicit pixel width (incl. narrow chrome / Actions) stays fixed.
  if (typeof col.width === 'number' && col.width > 0) return false;
  if (hasAuthorWidthSpec(getTableColumnMeta(col)?.widthSpec)) return false;
  const k = columnKey(col).toLowerCase();
  if (k === 'actions' || k.includes('__actions')) return false;
  // Narrow chrome without dataKey and without an explicit width (delete affordance).
  if (
    !col.dataKey &&
    typeof col.minWidth === 'number' &&
    col.minWidth > 0 &&
    col.minWidth <= 48
  ) {
    return false;
  }
  return true;
}

/**
 * Soft / semi-hard columns without an author widthSpec get a capped track.
 * Explicit numeric width stays fixed px. Author flex/auto/percent/capped wins.
 */
export function resolveColumnWidthSpec(
  col: Column,
  size: number,
): TableColumnMeta['widthSpec'] | undefined {
  const fromCol = getTableColumnMeta(col)?.widthSpec;
  if (hasAuthorWidthSpec(fromCol)) return fromCol;
  // Explicit pixel width → fixed track (no soft growth).
  if (typeof col.width === 'number' && col.width > 0) return undefined;
  if (isSystemListColumn(col)) return undefined;
  const fieldType = getTableColumnMeta(col)?.fieldType;
  // Cap from the author/type floor, not the post-sample size (avoids raising the
  // growth ceiling again after sampling already hit the original max).
  const baseMin =
    typeof col.minWidth === 'number' && col.minWidth > 0
      ? col.minWidth
      : FALLBACK_COLUMN_SIZE;
  const max = defaultListColumnMaxWidth(fieldType, baseMin, columnPropKey(col));
  if (max == null || max <= size) return undefined;
  return { type: 'capped', max };
}

/**
 * Stable list column order for grouped mode: selection → handle → index →
 * group label → remaining field columns (matches main OVTable normalizeColumns).
 */
export function orderListTableColumns(columns: readonly Column[]): Column[] {
  if (!columns.length) return [];
  const isSel = (c: Column) => {
    const k = columnKey(c);
    return k === '__selection__' || k === '__select' || c.type === 'selection';
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
 *
 * Soft (text-like) columns share leftover host width up to a type max; hard columns
 * stay fixed. Past sum(max), trailing gutter is left (no unbounded single-column fr).
 */
export function columnsToColumnDefs<T extends Record<string, unknown>>(
  columns: readonly Column[],
  options?: ColumnsToColumnDefsOptions,
): ColumnDef<T, unknown>[] {
  const ordered = orderListTableColumns(columns);
  const sampleRows = options?.sampleRows;
  const textMeasure = options?.textMeasure;
  const softMins =
    sampleRows && sampleRows.length > 0
      ? sampleSoftColumnMinWidths(
          ordered.map((col, index) => {
            const id = String(col.key ?? col.dataKey ?? `col_${index}`);
            const meta = getTableColumnMeta(col);
            const fieldType = meta?.fieldType;
            const explicitWidth =
              typeof col.width === 'number' && col.width > 0 ? col.width : undefined;
            const minWidth =
              typeof col.minWidth === 'number' && col.minWidth > 0
                ? col.minWidth
                : FALLBACK_COLUMN_SIZE;
            const maxWidth = defaultListColumnMaxWidth(fieldType, minWidth, columnPropKey(col));
            return {
              key: id,
              dataKey: col.dataKey != null ? String(col.dataKey) : undefined,
              title: col.title,
              fieldType,
              explicitWidth,
              minWidth,
              maxWidth,
              skipSample: hasAuthorWidthSpec(meta?.widthSpec) || explicitWidth != null,
            };
          }),
          sampleRows,
          {
            measure: textMeasure,
            rowLimit: options?.sampleRowLimit ?? LIST_SOFT_SAMPLE_ROW_LIMIT,
          },
        )
      : null;

  const userWidths = options?.userWidths;
  const indexDisplayMaxText = resolveIndexDisplayMaxText(options);

  return ordered.map((col, index) => {
    const id = String(col.key ?? col.dataKey ?? `col_${index}`);
    const userWidth = userWidths ? clampListColumnWidthPx(userWidths[id]) ?? undefined : undefined;
    const { size, minSize } = resolveColumnSizePx(col, {
      textMeasure,
      softSampleMin: softMins?.get(id),
      userWidth,
      indexDisplayMaxText,
    });
    const fromCol = getTableColumnMeta(col);
    const resizable = listColumnIsResizable(col);
    // User override forces a fixed track (leave soft share).
    const widthSpec =
      userWidth != null && resizable ? undefined : resolveColumnWidthSpec(col, size);

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
        // List host selection column (ChoyTableColumn type="selection"); key is the fallback.
        selectColumn: id === '__selection__' || id === '__select' || col.type === 'selection',
        resizable,
        userSized: userWidth != null && resizable,
        ...(widthSpec ? { widthSpec } : {}),
        ...(fromCol?.fieldType ? { fieldType: fromCol.fieldType } : {}),
      },
    };
    if (minSize != null) {
      def.minSize = minSize;
    }
    if (widthSpec?.type === 'capped') {
      def.maxSize = widthSpec.max;
    }
    if (col.dataKey != null) {
      (def as { accessorKey?: string }).accessorKey = String(col.dataKey);
    }
    return def;
  });
}
