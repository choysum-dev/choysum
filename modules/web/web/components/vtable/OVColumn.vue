<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Declares a column without rendering DOM directly. -->
</template>

<script setup lang="ts">
import { h, onMounted, onBeforeUnmount, useSlots, watch, inject } from 'vue';
import { ElCheckbox } from 'element-plus';
import type { Column } from 'element-plus';
import { useVTableUseColumnRegistry, useVTableUseBuildContext, setOVColumnMeta } from '@/web/web/composables/useVTable';
import type { OVColumnMeta } from '@/web/web/composables/useVTable';
import { LIST_HANDLE_API_KEY, type ListHandleReorderApi } from '@/web/web/composables/useListHandleReorder';

defineOptions({ name: 'OVColumn' });

let uid = 0;
const genKey = () => `__ovcol_${++uid}`;

const props = withDefaults(
  defineProps<{
    type?: 'default' | 'selection' | 'index' | 'handle';
    prop?: string;
    dataKey?: string;
    colKey?: string;
    label?: string;
    vColumnProps?: {
      width?: number | string; // Supports percentage and flex width semantics.
      minWidth?: number;
      align?: 'left' | 'center' | 'right';
      fixed?: 'left' | 'right';
      sortable?: boolean;
    };
    width?: number | string; // Supports percentage and flex width semantics.
    minWidth?: number;
    align?: 'left' | 'center' | 'right';
    fixed?: 'left' | 'right';
    sortable?: boolean;
  }>(),
  {
    type: 'default',
    label: '',
    vColumnProps: () => ({}),
  }
);

const slots = useSlots();
const reg = useVTableUseColumnRegistry();
const ctx = useVTableUseBuildContext();
const handleApi = inject<ListHandleReorderApi | null>(LIST_HANDLE_API_KEY, null);

let currentCol: Column | null = null;
let stableKey = props.colKey || props.dataKey || props.prop || genKey();

onMounted(() => registerOrReplace());
onBeforeUnmount(() => {
  if (!reg || !currentCol) return;
  const i = reg.columns.value.indexOf(currentCol);
  if (i >= 0) reg.columns.value.splice(i, 1);
  currentCol = null;
});

watch(
  () => ({
    type: props.type,
    prop: props.prop,
    dataKey: props.dataKey,
    colKey: props.colKey,
    label: props.label,
    width: props.width,
    minWidth: props.minWidth,
    align: props.align,
    fixed: props.fixed,
    sortable: props.sortable,
    vColumnProps: props.vColumnProps,
  }),
  () => registerOrReplace(),
  { deep: true }
);

function mergeColumnProps(raw: any) {
  const flat = {
    // Only pass numeric widths into Column; keep string widths in metadata.
    width: typeof raw?.width === 'number' ? raw?.width : undefined,
    minWidth: raw?.minWidth,
    align: raw?.align,
    fixed: raw?.fixed,
    sortable: raw?.sortable,
  };
  const base = raw?.vColumnProps || {};
  const flatClean = Object.fromEntries(Object.entries(flat).filter(([, v]) => v !== undefined));
  const baseClean = {
    ...base,
    ...(typeof base.width === 'number' ? { width: base.width } : {}), // Numeric widths only.
  };
  return { ...baseClean, ...flatClean };
}

function parseWidthSpec(
  w: unknown
): { type: 'percent'; ratio: number } | { type: 'flex'; weight: number } | { type: 'auto' } | { type: 'px'; value: number } | null {
  if (typeof w === 'number' && isFinite(w) && w > 0) return { type: 'px', value: w };
  if (typeof w !== 'string') return null;
  const s = w.trim().toLowerCase();
  if (!s) return null;
  if (s === 'auto') return { type: 'auto' };
  if (s.endsWith('%')) {
    const n = parseFloat(s.slice(0, -1));
    if (isFinite(n) && n > 0) return { type: 'percent', ratio: Math.min(Math.max(n / 100, 0), 1) };
  }
  // 1fr / 2fr / flex:2
  const fr = s.match(/^(\d+(?:\.\d+)?)fr$/);
  if (fr) return { type: 'flex', weight: Math.max(parseFloat(fr[1]), 0) || 1 };
  const fx = s.match(/^flex:(\d+(?:\.\d+)?)$/);
  if (fx) return { type: 'flex', weight: Math.max(parseFloat(fx[1]), 0) || 1 };
  return null;
}

function getWidthSpecFromProps(): ReturnType<typeof parseWidthSpec> {
  const p = props as any;
  return parseWidthSpec(p.width ?? p.vColumnProps?.width);
}

function getByPath(obj: any, path?: string) {
  if (!obj || !path) return undefined;
  const segs = String(path).split('.').filter(Boolean);
  let cur = obj;
  for (const s of segs) {
    if (cur == null) return undefined;
    cur = cur[s];
  }
  return cur;
}

function registerOrReplace() {
  if (!reg) return;
  const nextCol = buildColumn();
  if (!currentCol) {
    reg.columns.value.push(nextCol);
  } else {
    const i = reg.columns.value.indexOf(currentCol);
    if (i >= 0) reg.columns.value.splice(i, 1, nextCol);
    else reg.columns.value.push(nextCol);
  }
  currentCol = nextCol;
}

function buildColumn(): Column {
  const p = props as any;
  const colProps = mergeColumnProps(p);
  const type = p.type ?? 'default';
  const title = p.label ?? '';
  stableKey = p.colKey || stableKey;
  const dataKey = p.dataKey || p.prop || (type !== 'default' ? `__${type}__` : stableKey);

  const headerVNode = slots.header
    ? () => slots.header!({ label: title, prop: p.prop }) as any
    : () => h('span', null, type === 'index' && !title ? '#' : title);

  // Parse width semantics.
  const widthSpec = getWidthSpecFromProps();
  // Apply pixel widths immediately and let OVTable normalize the others.
  if (widthSpec?.type === 'px') colProps.width = widthSpec.value;

  // Build the Element Plus column definition.
  let col: Column;
  if (type === 'selection') {
    const width = colProps.width ?? 36;
    const api = ctx.selectionApi;
    const getRows = ctx.getRows ?? (() => []);
    const singleMode = api?.mode?.value === 'single';

    col = {
      key: dataKey,
      dataKey,
      title,
      width,
      align: 'center',
      sortable: false,
      ...colProps,
      headerCellRenderer: () => {
        if (singleMode) {
          // Hide the select-all checkbox in single-selection mode.
          return h('span', { style: 'display:inline-block;width:100%;height:1px;' });
        }
        const rows = getRows() || [];
        const allSel = rows.length > 0 && rows.every(r => api?.isSelected(r));
        const someSel = rows.some(r => api?.isSelected(r));
        return h(ElCheckbox, {
          modelValue: allSel,
          indeterminate: !allSel && someSel,
          'onUpdate:modelValue': (v: any) => api?.toggleAll(rows, !!v),
          onChange: (v: any) => api?.toggleAll(rows, !!v),
          onClick: (e: Event) => e.stopPropagation(),
        });
      },
      cellRenderer: ({ rowData }: any) =>
        h(ElCheckbox, {
          modelValue: api?.isSelected(rowData) ?? false,
          'onUpdate:modelValue': (v: any) => api?.toggleRow(rowData, !!v),
          onChange: (v: any) => api?.toggleRow(rowData, !!v),
          onClick: (e: Event) => e.stopPropagation(),
        }),
    };
  } else if (type === 'index') {
    const width = colProps.width ?? 50;
    col = {
      key: dataKey,
      dataKey,
      title: title || '#',
      width,
      align: 'right',
      sortable: false,
      ...colProps,
      headerCellRenderer: headerVNode,
      cellRenderer: ({ rowData, rowIndex }: any) => {
        // Group detail rows prefer the per-group index.
        const gi = (rowData as any)?.groupIndex;
        if (typeof gi === 'number') {
          return h('span', null, String(gi));
        }
        // Other rows fall back to the global row index with pagination offset.
        const raw = (ctx as any).baseIndex;
        const base = typeof raw === 'number' ? raw : Number(raw?.value ?? 1);
        return h('span', null, String(base + rowIndex));
      },
    };
  } else if (type === 'handle') {
    const width = colProps.width ?? 36;
    col = {
      key: dataKey,
      dataKey,
      title: title || '',
      width,
      align: 'center',
      sortable: false,
      ...colProps,
      headerCellRenderer: headerVNode,
      cellRenderer: ({ rowData, rowIndex }: any) => {
        const api = handleApi;
        const disabled = !api?.enabled?.value;
        return h(
          'span',
          {
            class: ['o-list-handle', disabled ? 'o-list-handle--disabled' : ''],
            draggable: disabled ? 'false' : 'true',
            title: disabled ? '' : 'Drag to reorder',
            onDragstart: (e: DragEvent) => {
              e.stopPropagation();
              api?.onDragStart(rowIndex, e);
            },
            onDragover: (e: DragEvent) => {
              e.stopPropagation();
              api?.onDragOver(rowIndex, e);
            },
            onDrop: (e: DragEvent) => {
              e.stopPropagation();
              void api?.onDrop(rowIndex, e);
            },
            onDragend: (e: DragEvent) => {
              e.stopPropagation();
              api?.onDragEnd();
            },
            onClick: (e: Event) => e.stopPropagation(),
          },
          [h('span', { class: 'o-list-handle__grip', 'aria-hidden': 'true' }, '⠿')]
        );
      },
    };
  } else {
    const cellSlot = typeof slots.default === 'function' ? slots.default : null;
    col = {
      key: dataKey,
      dataKey,
      title: title || '',
      ...colProps,
      headerCellRenderer: headerVNode,
      cellRenderer: ({ rowData, rowIndex, column }: any) => {
        if (cellSlot) return cellSlot({ row: rowData, column, $index: rowIndex, store: ctx.store }) as any;
        const path = p.prop || p.dataKey;
        let v = path ? getByPath(rowData, path) : undefined;

        // Grouped or aggregated rows may not carry raw field values, so fall back to metrics aliases.
        if (v == null && path && rowData) {
          const metrics = (rowData as any)?.metrics;
          if (metrics && typeof metrics === 'object') {
            // 1) Direct key match when the registered alias equals the field path.
            if (metrics[path] != null) v = metrics[path];
            // 2) Match a unique path__* alias or the special __count alias.
            if (v == null) {
              const candidates = Object.keys(metrics).filter(k => k === '__count' || k.startsWith(path + '__'));
              if (candidates.length === 1 && metrics[candidates[0]] != null) {
                v = metrics[candidates[0]];
              }
            }
          }
          // 3) Top-level mirrored aliases promoted by the executor, such as FinalAmount__sum.
          if (v == null) {
            const SUFFIXES = ['__sum', '__avg', '__min', '__max', '__count', '__count_distinct'];
            for (const suf of SUFFIXES) {
              const aliasKey = path + suf;
              if ((rowData as any)[aliasKey] != null) {
                v = (rowData as any)[aliasKey];
                break;
              }
            }
          }
        }

        const text = v == null ? '' : String(v);
        return h('span', { class: 'o-vcell__text', title: text }, text);
      },
    };
  }

  // Persist percentage, flex, and auto width semantics in metadata for OVTable.
  if (widthSpec && widthSpec.type !== 'px') {
    const meta: OVColumnMeta =
      widthSpec.type === 'percent'
        ? { widthSpec: { type: 'percent' as const, ratio: widthSpec.ratio } }
        : widthSpec.type === 'flex'
          ? { widthSpec: { type: 'flex' as const, weight: widthSpec.weight } }
          : { widthSpec: { type: 'auto' as const } };
    setOVColumnMeta(col, meta);
  } else if (typeof (col as any).width !== 'number') {
    // Treat unspecified non-pixel widths as auto as well.
    setOVColumnMeta(col, { widthSpec: { type: 'auto' as const } });
  }

  return col;
}
</script>

<style lang="scss" scoped>
:deep(.o-list-handle) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  user-select: none;
  cursor: grab;
}
:deep(.o-list-handle__grip) {
  font-size: 14px;
  line-height: 1;
  letter-spacing: -1px;
}
:deep(.o-list-handle--disabled) {
  opacity: 0.35;
  pointer-events: none;
  cursor: default;
}
</style>
