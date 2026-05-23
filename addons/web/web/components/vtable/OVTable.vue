<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-vtable">
    <!-- Hidden slot used only for column registration. -->
    <div class="ovtable__registrars" aria-hidden="true">
      <slot />
    </div>

    <el-auto-resizer v-slot="{ width, height }">
      <el-table-v2
        ref="tableRef"
        :key="width"
        :columns="normalizeColumns(columnsToUse, width)"
        :data="rowsArray"
        :row-key="rowKey"
        :width="width"
        :height="tableHeight ?? height"
        :row-height="rowHeight"
        :header-height="headerHeight"
        :footer-height="footerHeight"
        :row-event-handlers="rowEventHandlers"
        :on-scroll="handleScroll"
        scrollbar-always-on
        v-bind="$attrs"
      >
        <template #empty>
          <slot name="empty">
            <div class="ovtable__empty">暂无数据</div>
          </slot>
        </template>

        <template #footer>
          <slot name="footer" />
        </template>

        <!-- Row renderer that merges grouped rows into the group-label cell. -->
        <template #row="rp">
          <slot name="row" v-bind="rp">
            <RowMerge v-bind="rp" />
          </slot>
        </template>
      </el-table-v2>
    </el-auto-resizer>
  </div>
</template>

<script setup lang="ts">
import { computed, watch, Ref, ref, isRef, cloneVNode, h } from 'vue';
import { ElTableV2, ElAutoResizer, RowEventHandler } from 'element-plus';
import type { Column } from 'element-plus';
import { useVTableSelection, getOVColumnMeta, useVTableProvideColumnRegistry, useVTableProvideBuildContext } from '@/web/web/composables/useVTable';
import type { RowEventHandlerParams } from 'element-plus/es/components/table-v2/src/row';

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

// Normalize data input into an array.
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

// Compute a reactive base index, preferring store pagination when available.
const baseIndexRef = computed(() => {
  const p = (props.store as any)?.state?.pagination;
  if (p) return (p.currentPage - 1) * p.pageSize + 1;
  const bi = props.baseIndex as any;
  return typeof bi === 'number' ? bi : 1;
});

// Provide the shared build context.
useVTableProvideBuildContext({
  selectionApi: props.selectionApi,
  getRows: () => rowsArray.value,
  baseIndex: baseIndexRef,
  store: props.store,
});

// Column registry.
const { columns: regColumns } = useVTableProvideColumnRegistry();

// Resolve the active column source.
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

const tableRef = ref<InstanceType<typeof ElTableV2> | null>(null);

// Compute selection keys across both current and legacy row shapes.
// Prefer row.key, then __rowKey, then top-level Id/id, and finally payload.Id/id.
const keyOf = (row: any) =>
  row?.key ??
  row?.__rowKey ??
  row?.[props.rowKey!] ??
  row?.Id ??
  row?.id ??
  (typeof row === 'object' ? ((row as any)?.payload?.Id ?? (row as any)?.payload?.id) : undefined);

const selectedItems = computed(() => {
  const set = props.selectionApi?.selected.value ?? new Set<string | number>();
  const src = rowsArray.value || [];
  return src.filter(r => set.has(keyOf(r)));
});

watch(
  () => props.selectionApi?.selected.value,
  () => emit('selection-change', selectedItems.value)
);

// Public exposes.
function emitSortChange(payload: { field: string; direction?: 'asc' | 'desc' }) {
  emit('sort-change', payload);
}
defineExpose({
  selectedItems,
  emitSortChange,
  scrollTo: (left?: number, top?: number) => {
    // @ts-ignore
    tableRef.value?.scrollTo?.(left ?? 0, top ?? 0);
  },
  scrollToRow: (rowIndex: number, align?: 'auto' | 'smart' | 'center' | 'end' | 'start') => {
    // @ts-ignore
    tableRef.value?.scrollToRow?.(rowIndex, align ?? 'end');
  },
});

const rowEventHandlers: Record<string, RowEventHandler> = {
  onClick: p => emit('row-click', p),
  onContextmenu: p => emit('row-contextmenu', p),
  onDblclick: p => emit('row-dblclick', p),
  onMouseenter: p => emit('row-mouse-enter', p),
  onMouseleave: p => emit('row-mouse-leave', p),
};

function handleScroll(ev: any) {
  emit('scroll', ev);
}

/* ========= Functional row-merge component =========
  Keep the selection cell vnode intact.
  Move the group-label cell into the selection-cell position by copying left/transform,
  then null out the selection and index cells so the rendered content stays the group label. */
type RowSlotProps = { rowData: any; rowIndex: number; columns: any[]; cells: any[] };

const RowMerge = (rp: RowSlotProps) => {
  const { rowData, columns, cells } = rp;

  // Detail rows show the per-group index inside the index column.
  if ((rowData?.kind === 'record' || rowData?.type === 'record') && rowData?.groupIndex != null) {
    const getKey = (c: any) => c?.key ?? c?.dataKey;
    const idxCol = columns.findIndex(c => getKey(c) === '__index__');
    if (idxCol >= 0 && cells[idxCol]) {
      const style = (cells[idxCol]?.props?.style || {}) as any;
      const content = h('span', {}, String(rowData.groupIndex));
      cells[idxCol] = cloneVNode(cells[idxCol] as any, { style } as any, [content] as unknown as any) as any;
    }
    return cells;
  }

  // Merge "more" rows by moving the group cell to the first column and spanning the full row.
  if (rowData?.kind === 'more' || rowData?.type === 'more') {
    if (!cells?.length) return cells;
    const getKey = (c: any) => c?.key ?? c?.dataKey;
    const firstIdx = 0;
    const groupIdx = columns.findIndex(c => getKey(c) === '__group_label');
    if (groupIdx < 0) return cells;
    const px = (w: any) => Number.parseFloat(String(w || '0'));
    const totalWidth = cells.reduce((s, c) => s + px(c?.props?.style?.width), 0);
    const firstStyle = cells[firstIdx]?.props?.style || {};
    const gStyle = cells[groupIdx]?.props?.style || {};
    const style = {
      ...gStyle,
      left: firstStyle.left,
      transform: firstStyle.transform,
      width: `${totalWidth}px`,
    };
    cells[groupIdx] = cloneVNode(cells[groupIdx] as any, { style } as any);
    for (let i = 0; i < cells.length; i++) if (i !== groupIdx) cells[i] = null;
    return cells;
  }

  const isGroupRow = rowData?.kind === 'group' || rowData?.type === 'group';
  if (!isGroupRow) return cells;

  const getKey = (c: any) => c?.key ?? c?.dataKey;

  const selIdx = columns.findIndex(c => getKey(c) === '__selection__');
  const idxIdx = columns.findIndex(c => getKey(c) === '__index__');
  const grpIdx = columns.findIndex(c => getKey(c) === '__group_label');
  // Fallback: when no selection column exists, merge the first three columns into the group cell.
  if (grpIdx < 0) return cells;
  if (selIdx < 0) {
    if (!cells?.length) return cells;
    const firstSpan = Math.min(3, cells.length);
    if (firstSpan <= 0) return cells;
    const px = (w: any) => Number.parseFloat(String(w || '0'));
    // Sum the widths of the first merged columns.
    const totalWidth = cells.slice(0, firstSpan).reduce((s, c) => s + px(c?.props?.style?.width), 0);
    const firstStyle = cells[0]?.props?.style || {};
    const grpStyle = cells[grpIdx]?.props?.style || {};
    const newStyle = {
      ...grpStyle,
      left: firstStyle.left,
      transform: firstStyle.transform,
      width: `${Math.max(0, totalWidth)}px`,
    } as any;

    // Move the group cell to the first column and apply the merged width.
    const moved = cloneVNode(cells[grpIdx] as any, { style: newStyle } as any);
    // Clear the leading cells covered by the merge.
    for (let i = 0; i < firstSpan; i++) cells[i] = null;
    cells[0] = moved as any;
    // Clear the original group-cell slot as well when it falls outside the merged prefix.
    if (grpIdx >= firstSpan) cells[grpIdx] = null as any;
    return cells;
  }
  // Normal path: merge the selection, index, and group-label columns.

  // Compute the merged width.
  const px = (w: any) => Number.parseFloat(String(w || '0'));
  const selW = px(cells[selIdx]?.props?.style?.width);
  const idxW = idxIdx >= 0 ? px(cells[idxIdx]?.props?.style?.width) : 0;
  const grpW = px(cells[grpIdx]?.props?.style?.width);
  const mergedWidth = selW + idxW + grpW;

  // Move the group cell into the selection-cell slot and stretch its width.
  const selStyle = cells[selIdx]?.props?.style || {};
  const grpStyle = cells[grpIdx]?.props?.style || {};
  const newStyle = {
    ...grpStyle,
    left: selStyle.left,
    transform: selStyle.transform,
    width: `${mergedWidth}px`,
  };

  cells[grpIdx] = cloneVNode(cells[grpIdx], { style: newStyle });
  // Hide the merged selection and index cells.
  cells[selIdx] = null;
  if (idxIdx > -1) cells[idxIdx] = null;

  return cells;
};
/* ===================================== */

function normalizeColumns(cols: Column[], containerWidth: number): Column[] {
  if (!Array.isArray(cols) || !containerWidth) return cols;

  // 1) Reorder columns as selection -> index -> group label -> the remaining columns.
  const getKey = (c: any) => c?.key ?? c?.dataKey;
  const isSel = (c: any) => getKey(c) === '__selection__';
  const isIdx = (c: any) => getKey(c) === '__index__';
  const isGroupLabel = (c: any) => getKey(c) === '__group_label';

  const selCols = cols.filter(isSel);
  const idxCols = cols.filter(isIdx);
  const groupCols = cols.filter(isGroupLabel);
  const restCols = cols.filter(c => !selCols.includes(c) && !idxCols.includes(c) && !groupCols.includes(c));

  const ordered = [...selCols, ...idxCols, ...groupCols, ...restCols];

  // 2) Clone columns before width normalization.
  const out = ordered.map(c => ({ ...c })) as Column[];

  // 3) Reuse the existing width-normalization logic.
  let used = 0;
  out.forEach(c => {
    if (typeof c.width === 'number' && isFinite(c.width)) used += c.width!;
  });

  out.forEach(c => {
    const meta = getOVColumnMeta(c);
    if (meta?.widthSpec && meta.widthSpec.type === 'percent') {
      const w = Math.floor(containerWidth * meta.widthSpec.ratio);
      const min = (c as any).minWidth ?? 120;
      (c as any).width = Math.max(min, w);
      used += (c as any).width;
    }
  });

  const remain = Math.max(0, containerWidth - used);
  const flexCols: Array<{ c: any; weight: number; min: number }> = [];

  out.forEach(c => {
    const meta = getOVColumnMeta(c);
    const hasFixed = typeof c.width === 'number' && isFinite(c.width);
    if (hasFixed) return;
    const min = (c as any).minWidth ?? 120;
    let weight = 1;
    if (meta?.widthSpec?.type === 'flex') weight = Math.max(meta.widthSpec.weight || 1, 0.0001);
    flexCols.push({ c, weight, min });
  });

  const totalWeight = flexCols.reduce((s, f) => s + f.weight, 0) || 1;
  flexCols.forEach((f, idx) => {
    let w = Math.floor((remain * f.weight) / totalWeight);
    w = Math.max(f.min, w);
    if (idx === flexCols.length - 1) {
      const current = out.reduce((s, c) => s + (typeof (c as any).width === 'number' ? (c as any).width : 0), 0);
      w = Math.max(f.min, containerWidth - current);
    }
    (f.c as any).width = w;
  });

  return out;
}
</script>

<style lang="scss" scoped>
.o-vtable {
  width: 100%;
  height: 100%;
  min-width: 0;
  overscroll-behavior: contain;
  touch-action: pan-y;
}
.ovtable__registrars {
  display: none;
}
.ovtable__empty {
  width: 100%;
  padding: 24px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}
:deep(.el-auto-resizer) {
  width: 100%;
  height: 100%;
  min-width: 0;
}
</style>
