<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :preserveModeSlot="true"
    :showInlineError="showInlineError"
  >
    <template #edit>
      <div class="o-m2m-tree" :style="treeBoxStyle">
        <el-tree
          ref="editTreeRef"
          class="o-m2m-tree__tree"
          node-key="__id"
          :data="treeData"
          :props="treeProps"
          :lazy="lazy"
          :load="loadTreeNode"
          :show-checkbox="true"
          :check-strictly="checkStrictly"
          :default-expand-all="defaultExpandAll"
          :expand-on-click-node="false"
          :highlight-current="false"
          :empty-text="effectiveEmptyText"
          :loading="loading"
          @check="onCheck"
          @node-click="onNodeClick"
        >
          <template #default="{ node, data }">
            <slot name="node-edit" :row="data?.raw" :label="data?.__label" :id="data?.__id" mode="edit" :treeNode="node">
              <slot name="node" :row="data?.raw" :label="data?.__label" :id="data?.__id" mode="edit" :treeNode="node">
                <span>{{ data?.__label }}</span>
              </slot>
            </slot>
          </template>
        </el-tree>
      </div>
    </template>

    <template #display>
      <div class="o-m2m-tree o-m2m-tree--readonly" :style="treeBoxStyle" @click.capture="onDisplayTreeClickCapture">
        <el-tree
          ref="displayTreeRef"
          class="o-m2m-tree__tree"
          node-key="__id"
          :data="treeData"
          :props="treeProps"
          :lazy="lazy"
          :load="loadTreeNode"
          :show-checkbox="true"
          :check-on-click-node="false"
          :check-on-click-leaf="false"
          :check-strictly="checkStrictly"
          :default-expand-all="defaultExpandAll"
          :expand-on-click-node="false"
          :highlight-current="false"
          :empty-text="effectiveEmptyText"
          :loading="loading"
          @node-click="onNodeClick"
        >
          <template #default="{ node, data }">
            <slot name="node-display" :row="data?.raw" :label="data?.__label" :id="data?.__id" mode="display" :treeNode="node">
              <slot name="node" :row="data?.raw" :label="data?.__label" :id="data?.__id" mode="display" :treeNode="node">
                <span>{{ data?.__label }}</span>
              </slot>
            </slot>
          </template>
        </el-tree>
      </div>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, watch, inject, type Ref } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, ClientModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps, TreeInstance } from 'element-plus';
import { ElTree } from 'element-plus';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import { createStoreByModel } from '@/web/web/stores/registry';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OManyToManyTreeField' });

defineOptions({ name: 'OManyToManyTreeField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;
type ConditionLeaf = [field: string, op: string, value: unknown];
type AnyQueryCondition = ConditionLeaf | { And: AnyQueryCondition[] } | { Or: AnyQueryCondition[] };
type TreeNode = {
  __id: string;
  __label: string;
  __order: number;
  __depth: number;
  __leafKnown?: boolean;
  __childrenLoaded?: boolean;
  raw: Record<string, any>;
  children?: TreeNode[];
};

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;

    targetModel?: string;
    childrenField?: string;
    nodeKeyField?: string;
    labelField?: string | string[];
    orderField?: string;
    fields?: string[];
    fetchLimit?: number;
    lazy?: boolean;
    maxDepth?: number;
    rootCondition?: AnyQueryCondition | AnyQueryCondition[];

    checkStrictly?: boolean;
    defaultExpandAll?: boolean;
    expandOnClickNode?: boolean;
    emptyText?: string;
    treeHeight?: number | string;

    condition?: AnyQueryCondition | AnyQueryCondition[];

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    label: '',
    rules: () => [],
    formItemProps: () => ({}),
    targetModel: '',
    childrenField: 'Childs',
    nodeKeyField: 'Id',
    labelField: () => ['Title', 'Name', 'DisplayName', 'Id'],
    orderField: 'Sequence',
    fields: () => [],
    fetchLimit: 5000,
    lazy: false,
    maxDepth: 0,
    rootCondition: undefined,
    checkStrictly: true,
    defaultExpandAll: false,
    expandOnClickNode: true,
    emptyText: '',
    treeHeight: 360,
    condition: undefined,
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const effectiveEmptyText = computed(() => props.emptyText || _t('No data'));

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;
const { getItems, clearItems, insertItem } = binding.asMutableArray<any>();

const relationStore = computed<WebModelStore<any> | undefined>(() => {
  if (binding.relationStore) return binding.relationStore as WebModelStore<any>;
  const target = props.targetModel || binding.meta?.relationModel;
  if (!target) return undefined;
  try {
    return createStoreByModel(target);
  } catch (e) {
    console.warn(`[OManyToManyTreeField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});

const treeData = ref<TreeNode[]>([]);
const loading = ref(false);
const editTreeRef = ref<TreeInstance | null>(null);
const displayTreeRef = ref<TreeInstance | null>(null);
const syncingTree = ref(false);
let loadVersion = 0;

const treeProps = {
  label: '__label',
  children: 'children',
  isLeaf: '__leafKnown',
} as const;

const treeBoxStyle = computed(() => {
  const h = typeof props.treeHeight === 'number' ? `${props.treeHeight}px` : String(props.treeHeight || '360px');
  return { height: h };
});

const labelFields = computed<string[]>(() => {
  const raw = props.labelField;
  const list = Array.isArray(raw) ? raw : [raw];
  return list.map(x => String(x || '').trim()).filter(Boolean);
});

const normalizedMaxDepth = computed<number>(() => {
  const n = Number(props.maxDepth ?? 0);
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.floor(n);
});

const baseLoadFields = computed<string[]>(() => {
  const set = new Set<string>(['Id']);
  set.add(String(props.nodeKeyField || 'Id'));
  set.add(String(props.orderField || 'Sequence'));
  for (const f of labelFields.value) set.add(f);
  for (const f of props.fields || []) {
    const key = String(f || '').trim();
    if (key) set.add(key);
  }
  return Array.from(set);
});

const perRequestLimit = computed<number>(() => {
  const n = Number(props.fetchLimit ?? 5000);
  if (!Number.isFinite(n) || n <= 0) return 5000;
  return Math.floor(n);
});

const parentQueryFields = computed<any[]>(() => {
  const relationName = String(props.childrenField || 'Childs');
  return ['Id', relationName] as any[];
});

const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));
function toArray<TX>(v: TX | TX[] | undefined | null): TX[] {
  if (v == null) return [];
  return Array.isArray(v) ? v : [v];
}

const externalConditions = computed<AnyQueryCondition[]>(() => toArray(props.condition));
const fieldName = computed(() => String(binding.prop));
const onchangeConditions = computed<AnyQueryCondition[]>(() => {
  const raw = lastOnchangeResult.value?.condition || [];
  return raw
    .filter((c: any) => c?.field === fieldName.value)
    .map((c: any) => c?.condition)
    .filter(Boolean);
});

const effectiveRootCondition = computed<AnyQueryCondition | []>(() => {
  const parts: AnyQueryCondition[] = [];
  parts.push(...toArray(props.rootCondition), ...externalConditions.value, ...onchangeConditions.value);
  if (!parts.length) return [] as any;
  if (parts.length === 1) return parts[0];
  return { And: parts } as any;
});

function normalizeRefId(v: any): string | null {
  if (v == null) return null;
  if (typeof v === 'object' && v !== null && typeof (v as any).toEntity === 'function') {
    const entity = (v as any).toEntity();
    const s = String(entity?.Id ?? '').trim();
    if (s) return s;
  }
  const raw = typeof v === 'object' && v !== null ? (v as any).Id : v;
  const s = String(raw ?? '').trim();
  return s ? s : null;
}

function parseChildrenValue(raw: any): any[] {
  if (Array.isArray(raw)) return raw;

  if (typeof raw === 'string') {
    const s = raw.trim();
    if (!s) return [];
    if (s[0] !== '[' && s[0] !== '{') return [];
    try {
      const parsed = JSON.parse(s);
      return Array.isArray(parsed) ? parsed : [];
    } catch {
      return [];
    }
  }

  if (!raw || typeof raw !== 'object') return [];
  for (const key of ['value', 'values', 'items']) {
    const v = (raw as any)[key];
    if (Array.isArray(v)) return v;
  }
  return [];
}

function normalizeRow(input: any): Record<string, any> {
  if (!input || typeof input !== 'object') return {};
  if (typeof (input as any).toEntity === 'function') {
    const entity = (input as any).toEntity();
    return entity && typeof entity === 'object' ? (entity as Record<string, any>) : {};
  }
  return input as Record<string, any>;
}

function readByFieldName(row: Record<string, any>, fieldName: string): any {
  return (row as any)?.[fieldName];
}

function resolveLabel(row: Record<string, any>): string {
  for (const f of labelFields.value) {
    const val = readByFieldName(row, f);
    const s = String(val ?? '').trim();
    if (s) return s;
  }
  return String(normalizeRefId(readByFieldName(row, props.nodeKeyField)) || '');
}

function toNodeRow(input: any, depth: number): TreeNode | null {
  const row = normalizeRow(input);
  if (!row || typeof row !== 'object') return null;
  const nodeId = normalizeRefId(readByFieldName(row, props.nodeKeyField));
  if (!nodeId) return null;

  const orderVal = Number(readByFieldName(row, props.orderField));
  const isAtMaxDepth = normalizedMaxDepth.value > 0 && depth >= normalizedMaxDepth.value;
  return {
    __id: nodeId,
    __label: resolveLabel(row),
    __order: Number.isFinite(orderVal) ? orderVal : 0,
    __depth: depth,
    __leafKnown: isAtMaxDepth ? true : undefined,
    __childrenLoaded: isAtMaxDepth,
    raw: row,
    children: isAtMaxDepth ? [] : undefined,
  };
}

function sortNodes(nodes: TreeNode[]): void {
  nodes.sort((a, b) => {
    if (a.__order !== b.__order) return a.__order - b.__order;
    return a.__label.localeCompare(b.__label);
  });
  for (const node of nodes) {
    if (Array.isArray(node.children) && node.children.length) {
      sortNodes(node.children);
    }
  }
}

function readChildrenRows(row: Record<string, any>): any[] {
  const relationName = String(props.childrenField || 'Childs');
  const raw = readByFieldName(row, relationName);
  return parseChildrenValue(raw);
}

async function fetchRawChildrenByParentIds(parentIds: string[]): Promise<Map<string, any[]>> {
  const result = new Map<string, any[]>();
  const ids = Array.from(new Set((parentIds || []).map(id => String(id || '').trim()).filter(Boolean)));
  for (const id of ids) result.set(id, []);
  if (!ids.length) return result;

  const store = relationStore.value;
  if (!store) return result;

  const childSearchCondition = {
    And: [['Id', 'in', ids]],
  } as any;

  const childSearchOptions = {
    fields: parentQueryFields.value as any,
    limit: Math.max(perRequestLimit.value, ids.length),
  } as any;

  const rows = await store.Search(childSearchCondition, childSearchOptions);

  const responseRows = Array.isArray(rows) ? rows : [];

  for (const rowInput of responseRows) {
    const row = normalizeRow(rowInput);
    const id = normalizeRefId(readByFieldName(row, 'Id'));
    if (!id) continue;
    result.set(id, readChildrenRows(row));
  }

  return result;
}

async function loadChildrenForNode(node: TreeNode): Promise<TreeNode[]> {
  if (node.__childrenLoaded && Array.isArray(node.children)) {
    return node.children;
  }

  if (normalizedMaxDepth.value > 0 && node.__depth >= normalizedMaxDepth.value) {
    node.children = [];
    node.__childrenLoaded = true;
    node.__leafKnown = true;
    return [];
  }

  const childrenMap = await fetchRawChildrenByParentIds([node.__id]);
  const children = (childrenMap.get(node.__id) || []).map(item => toNodeRow(item, node.__depth + 1)).filter(Boolean) as TreeNode[];

  sortNodes(children);
  node.children = children;
  node.__childrenLoaded = true;
  node.__leafKnown = children.length === 0;
  return children;
}

async function preloadTreeByLayers(roots: TreeNode[]): Promise<void> {
  if (!roots.length) return;
  if (normalizedMaxDepth.value > 0 && normalizedMaxDepth.value <= 1) return;

  let frontier = roots;
  while (frontier.length > 0) {
    const expandable = frontier.filter(node => normalizedMaxDepth.value === 0 || node.__depth < normalizedMaxDepth.value);
    if (!expandable.length) break;

    const childrenMap = await fetchRawChildrenByParentIds(expandable.map(node => node.__id));
    const nextFrontier: TreeNode[] = [];

    for (const node of expandable) {
      const children = (childrenMap.get(node.__id) || []).map(item => toNodeRow(item, node.__depth + 1)).filter(Boolean) as TreeNode[];

      sortNodes(children);
      node.children = children;
      node.__childrenLoaded = true;
      node.__leafKnown = children.length === 0;

      for (const child of children) {
        if (normalizedMaxDepth.value > 0 && child.__depth >= normalizedMaxDepth.value) {
          child.children = [];
          child.__childrenLoaded = true;
          child.__leafKnown = true;
          continue;
        }
        nextFrontier.push(child);
      }
    }

    if (!nextFrontier.length) break;
    frontier = nextFrontier;
  }
}

function readCurrentSelectedIds(): string[] {
  return Array.from(
    new Set(
      (getItems() || [])
        .map(it => normalizeRefId(it))
        .filter(Boolean)
        .map(String)
    )
  );
}

function collectNodeRawByIdFromTree(nodes: TreeNode[]): Map<string, Record<string, any>> {
  const result = new Map<string, Record<string, any>>();
  const stack = [...nodes];

  while (stack.length) {
    const node = stack.pop();
    if (!node) continue;
    if (node.__id && node.raw) {
      result.set(node.__id, node.raw);
    }
    if (Array.isArray(node.children) && node.children.length) {
      stack.push(...node.children);
    }
  }

  return result;
}

function applySelectedIds(ids: string[]): void {
  const currentItems = getItems() || [];
  const currentById = new Map<string, any>();
  for (const item of currentItems) {
    const id = normalizeRefId(item);
    if (!id) continue;
    currentById.set(id, item);
  }

  const nodeRawById = collectNodeRawByIdFromTree(treeData.value);

  clearItems();
  for (const id of ids) {
    const existing = currentById.get(id);
    if (existing && typeof existing === 'object') {
      insertItem(existing as any);
      continue;
    }

    const raw = nodeRawById.get(id);
    if (raw && typeof raw === 'object') {
      insertItem(raw as any);
      continue;
    }

    insertItem({ Id: id } as any);
  }
}

function syncTreeCheckedKeys(): void {
  if (syncingTree.value) return;
  syncingTree.value = true;
  try {
    const ids = readCurrentSelectedIds();
    editTreeRef.value?.setCheckedKeys(ids, false);
    displayTreeRef.value?.setCheckedKeys(ids, false);
  } finally {
    syncingTree.value = false;
  }
}

async function loadTreeRows() {
  const store = relationStore.value;
  if (!store) {
    treeData.value = [];
    return;
  }

  const version = ++loadVersion;
  loading.value = true;
  try {
    const rootSearchOptions = {
      fields: baseLoadFields.value as any,
      limit: perRequestLimit.value,
    } as any;

    const rows = await store.Search(effectiveRootCondition.value as any, rootSearchOptions);

    const responseRows = Array.isArray(rows) ? rows : [];

    const roots = responseRows.map(row => toNodeRow(row, 1)).filter(Boolean) as TreeNode[];
    sortNodes(roots);

    if (!props.lazy) {
      await preloadTreeByLayers(roots);
    }

    if (version !== loadVersion) return;
    treeData.value = roots;
    syncTreeCheckedKeys();
  } catch (e) {
    console.warn('[OManyToManyTreeField] load tree rows failed', e);
    treeData.value = [];
  } finally {
    if (version === loadVersion) {
      loading.value = false;
    }
  }
}

async function loadTreeNode(node: any, resolve: (data: TreeNode[]) => void): Promise<void> {
  if (!props.lazy) {
    resolve([]);
    return;
  }

  if (!node || Number(node.level || 0) === 0) {
    resolve(treeData.value);
    return;
  }

  const data = node.data as TreeNode | undefined;
  if (!data) {
    resolve([]);
    return;
  }

  try {
    const children = await loadChildrenForNode(data);
    resolve(children);
    syncTreeCheckedKeys();
  } catch (e) {
    console.warn('[OManyToManyTreeField] lazy load node children failed', e);
    data.children = [];
    data.__childrenLoaded = true;
    data.__leafKnown = true;
    resolve([]);
  }
}

function onCheck(_data: any, info: any) {
  if (syncingTree.value) return;
  const checkedKeysFromTree = editTreeRef.value?.getCheckedKeys(false);
  const sourceKeys = Array.isArray(checkedKeysFromTree) ? checkedKeysFromTree : Array.isArray(info?.checkedKeys) ? info.checkedKeys : [];
  const ids: string[] = Array.from(new Set(sourceKeys.map((x: any) => String(x))));
  applySelectedIds(ids);
  syncTreeCheckedKeys();
}

function toggleNodeExpanded(node: any) {
  if (!node) return;
  if (node.expanded) {
    if (typeof node.collapse === 'function') {
      node.collapse();
      return;
    }
    node.expanded = false;
    return;
  }

  if (typeof node.expand === 'function') {
    node.expand();
    return;
  }
  node.expanded = true;
}

function onNodeClick(_data: any, node: any, _treeNode: any, event?: MouseEvent) {
  if (!props.expandOnClickNode) return;
  const target = event?.target as HTMLElement | null;
  if (target?.closest('.el-checkbox')) return;
  toggleNodeExpanded(node);
}

function onDisplayTreeClickCapture(event: MouseEvent) {
  const target = event.target as HTMLElement | null;
  if (!target?.closest('.el-checkbox')) return;
  event.preventDefault();
  event.stopPropagation();
}

watch(
  () =>
    [
      relationStore.value,
      effectiveRootCondition.value,
      baseLoadFields.value.join('|'),
      props.childrenField,
      props.nodeKeyField,
      props.orderField,
      props.lazy,
      normalizedMaxDepth.value,
    ] as const,
  () => {
    void loadTreeRows();
  },
  { immediate: true, deep: true }
);

watch(
  () => getItems(),
  () => syncTreeCheckedKeys(),
  { deep: true }
);
</script>

<style scoped>
.o-m2m-tree {
  width: 100%;
  padding: 8px;
  overflow: auto;
}

.o-m2m-tree__tree {
  min-height: 120px;
}

.o-m2m-tree--readonly :deep(.el-checkbox) {
  cursor: default;
}
</style>
