<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.internal.relation-tree"
    :class="cn('choy-relation-tree', props.class)"
  >
    <div
      v-if="loading && !visibleRows.length"
      class="choy-relation-tree__status px-2 py-3 text-sm text-foreground/60"
    >
      Loading…
    </div>
    <div
      v-else-if="!visibleRows.length"
      class="choy-relation-tree__status px-2 py-3 text-sm text-foreground/60"
    >
      {{ emptyText || 'No data' }}
    </div>
    <ul v-else class="choy-relation-tree__list">
      <li
        v-for="row in visibleRows"
        :key="row.key"
        class="choy-relation-tree__item"
        :data-node-key="row.key"
      >
        <div
          class="choy-relation-tree__row flex items-center gap-1 py-0.5 text-sm"
          :style="{ paddingLeft: `${row.depth * 16}px` }"
          @click="onRowClick(row, $event)"
        >
          <button
            v-if="!row.isLeaf"
            type="button"
            class="choy-relation-tree__expand inline-flex size-5 shrink-0 items-center justify-center rounded-sm text-foreground/70 hover:bg-muted"
            :aria-expanded="row.expanded ? 'true' : 'false'"
            @click.stop="toggleExpanded(row.key)"
          >
            <span class="choy-relation-tree__expand-icon inline-block leading-none transition-transform duration-150 ease-in-out" :class="{ 'choy-relation-tree__expand-icon--open rotate-90': row.expanded }">
              ›
            </span>
          </button>
          <span v-else class="inline-block size-5 shrink-0" aria-hidden="true" />
          <Checkbox
            v-if="showCheckbox"
            class="choy-relation-tree__checkbox"
            :model-value="checkedSet.has(row.key)"
            :disabled="checkboxDisabled"
            @update:model-value="(v) => onCheckboxChange(row, v === true)"
            @click.stop
          />
          <div class="choy-relation-tree__label min-w-0 flex-1">
            <slot name="default" :node="toTreeNodeProxy(row)" :data="row.data">
              {{ row.label }}
            </slot>
          </div>
          <span
            v-if="row.loadingChildren"
            class="choy-relation-tree__node-loading shrink-0 text-xs text-foreground/50"
          >
            …
          </span>
        </div>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import Checkbox from '../vendor/ui/checkbox/Checkbox.vue';

export type RelationTreePropsMap = {
  label?: string;
  children?: string;
  isLeaf?: string;
};

export type RelationTreeLoadFn = (
  node: RelationTreeNodeProxy,
  resolve: (children: any[]) => void,
) => void | Promise<void>;

/** Minimal el-tree-like node surface used by relation tree fields. */
export type RelationTreeNodeProxy = {
  level: number;
  data: any;
  expanded: boolean;
  expand: () => void;
  collapse: () => void;
};

type VisibleRow = {
  key: string;
  data: any;
  label: string;
  depth: number;
  level: number;
  expanded: boolean;
  isLeaf: boolean;
  loadingChildren: boolean;
};

/**
 * Checkbox tree for relation M2M pickers (lazy roots/children, check-strictly).
 * Not a public Choy* export — hosts wire FieldBase slots around the default slot.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    data?: any[];
    nodeKey?: string;
    props?: RelationTreePropsMap;
    lazy?: boolean;
    load?: RelationTreeLoadFn;
    showCheckbox?: boolean;
    checkStrictly?: boolean;
    defaultExpandAll?: boolean;
    expandOnClickNode?: boolean;
    emptyText?: string;
    loading?: boolean;
    /** When true, checkboxes render current state but cannot be toggled. */
    checkboxDisabled?: boolean;
  }>(),
  {
    data: () => [],
    nodeKey: '__id',
    props: () => ({ label: '__label', children: 'children', isLeaf: '__leafKnown' }),
    lazy: false,
    showCheckbox: true,
    checkStrictly: true,
    defaultExpandAll: false,
    expandOnClickNode: false,
    emptyText: '',
    loading: false,
    checkboxDisabled: false,
  },
);

const emit = defineEmits<{
  check: [data: any, info: { checkedKeys: string[]; checkedNodes: any[] }];
  'node-click': [data: any, node: RelationTreeNodeProxy, treeNode: null, event: MouseEvent];
}>();

const labelKey = computed(() => String(props.props?.label || '__label'));
const childrenKey = computed(() => String(props.props?.children || 'children'));
const isLeafKey = computed(() => String(props.props?.isLeaf || '__leafKnown'));
const keyField = computed(() => String(props.nodeKey || '__id'));

const expandedSet = reactive(new Set<string>());
const checkedSet = reactive(new Set<string>());
const loadingChildKeys = reactive(new Set<string>());
/** Children injected by lazy `load` when the host has not written them onto `data` yet. */
const lazyChildren = ref<Record<string, any[]>>({});

function nodeKeyOf(data: any): string {
  const raw = data?.[keyField.value];
  return raw == null ? '' : String(raw);
}

function nodeLabelOf(data: any): string {
  const raw = data?.[labelKey.value];
  return raw == null ? nodeKeyOf(data) : String(raw);
}

function childrenOf(data: any): any[] | undefined {
  const key = nodeKeyOf(data);
  if (key && Object.prototype.hasOwnProperty.call(lazyChildren.value, key)) {
    return lazyChildren.value[key];
  }
  const raw = data?.[childrenKey.value];
  return Array.isArray(raw) ? raw : undefined;
}

function isLeafOf(data: any): boolean {
  const flag = data?.[isLeafKey.value];
  if (flag === true) return true;
  if (flag === false) return false;
  const kids = childrenOf(data);
  if (Array.isArray(kids)) return kids.length === 0 && !props.lazy;
  return false;
}

function collectKeys(nodes: any[] | undefined, out: string[]): void {
  if (!Array.isArray(nodes)) return;
  for (const n of nodes) {
    const k = nodeKeyOf(n);
    if (k) out.push(k);
    collectKeys(childrenOf(n), out);
  }
}

function expandAllFromData(): void {
  const keys: string[] = [];
  collectKeys(props.data, keys);
  expandedSet.clear();
  for (const k of keys) expandedSet.add(k);
}

watch(
  () => [props.data, props.defaultExpandAll] as const,
  () => {
    lazyChildren.value = {};
    if (props.defaultExpandAll) {
      expandAllFromData();
    }
  },
  { immediate: true, deep: true },
);

const visibleRows = computed<VisibleRow[]>(() => {
  const out: VisibleRow[] = [];
  const walk = (nodes: any[] | undefined, depth: number) => {
    if (!Array.isArray(nodes)) return;
    for (const data of nodes) {
      const key = nodeKeyOf(data);
      if (!key) continue;
      const expanded = expandedSet.has(key);
      const kids = childrenOf(data);
      const leaf = isLeafOf(data);
      out.push({
        key,
        data,
        label: nodeLabelOf(data),
        depth,
        level: depth + 1,
        expanded,
        isLeaf: leaf,
        loadingChildren: loadingChildKeys.has(key),
      });
      if (expanded && Array.isArray(kids) && kids.length) {
        walk(kids, depth + 1);
      }
    }
  };
  walk(props.data, 0);
  return out;
});

function toTreeNodeProxy(row: VisibleRow): RelationTreeNodeProxy {
  return {
    level: row.level,
    data: row.data,
    get expanded() {
      return expandedSet.has(row.key);
    },
    set expanded(v: boolean) {
      if (v) expandedSet.add(row.key);
      else expandedSet.delete(row.key);
    },
    expand: () => {
      void expandKey(row.key, row.data, row.level);
    },
    collapse: () => {
      expandedSet.delete(row.key);
    },
  };
}

async function expandKey(key: string, data: any, level: number): Promise<void> {
  if (expandedSet.has(key)) return;
  expandedSet.add(key);

  const existing = childrenOf(data);
  if (Array.isArray(existing)) return;
  if (!props.lazy || typeof props.load !== 'function') {
    return;
  }
  if (loadingChildKeys.has(key)) return;

  loadingChildKeys.add(key);
  try {
    const proxy = {
      level,
      data,
      expanded: true,
      expand: () => {},
      collapse: () => {
        expandedSet.delete(key);
      },
    } satisfies RelationTreeNodeProxy;
    const children = await new Promise<any[]>((resolve) => {
      void Promise.resolve(props.load!(proxy, (rows) => resolve(Array.isArray(rows) ? rows : [])));
    });
    lazyChildren.value = { ...lazyChildren.value, [key]: children };
    if (Array.isArray(data) === false && data && typeof data === 'object') {
      // Prefer host-owned children when the load path mutates the node.
      if (!Array.isArray(data[childrenKey.value])) {
        data[childrenKey.value] = children;
      }
    }
  } finally {
    loadingChildKeys.delete(key);
  }
}

function toggleExpanded(key: string): void {
  const row = visibleRows.value.find((r) => r.key === key);
  if (!row) return;
  if (expandedSet.has(key)) {
    expandedSet.delete(key);
    return;
  }
  void expandKey(key, row.data, row.level);
}

function onRowClick(row: VisibleRow, event: MouseEvent): void {
  const proxy = toTreeNodeProxy(row);
  emit('node-click', row.data, proxy, null, event);
  if (!props.expandOnClickNode) return;
  const target = event.target as HTMLElement | null;
  if (target?.closest('.choy-relation-tree__checkbox')) return;
  toggleExpanded(row.key);
}

function emitCheck(data: any): void {
  const checkedKeys = Array.from(checkedSet);
  const checkedNodes: any[] = [];
  const walk = (nodes: any[] | undefined) => {
    if (!Array.isArray(nodes)) return;
    for (const n of nodes) {
      const k = nodeKeyOf(n);
      if (k && checkedSet.has(k)) checkedNodes.push(n);
      walk(childrenOf(n));
    }
  };
  walk(props.data);
  emit('check', data, { checkedKeys, checkedNodes });
}

function onCheckboxChange(row: VisibleRow, checked: boolean): void {
  if (props.checkboxDisabled) return;
  if (checked) checkedSet.add(row.key);
  else checkedSet.delete(row.key);

  // checkStrictly=false would cascade; relation fields use strict mode.
  if (!props.checkStrictly) {
    const kids = childrenOf(row.data) || [];
    const cascade = (nodes: any[]) => {
      for (const n of nodes) {
        const k = nodeKeyOf(n);
        if (!k) continue;
        if (checked) checkedSet.add(k);
        else checkedSet.delete(k);
        cascade(childrenOf(n) || []);
      }
    };
    cascade(kids);
  }
  emitCheck(row.data);
}

function setCheckedKeys(keys: string[] | null | undefined, _leafOnly?: boolean): void {
  checkedSet.clear();
  for (const k of keys || []) {
    const id = String(k || '').trim();
    if (id) checkedSet.add(id);
  }
}

function getCheckedKeys(_leafOnly?: boolean): string[] {
  return Array.from(checkedSet);
}

defineExpose({
  setCheckedKeys,
  getCheckedKeys,
});
</script>

