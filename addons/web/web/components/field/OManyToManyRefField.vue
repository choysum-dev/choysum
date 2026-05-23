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
    :showInlineError="showInlineError"
  >
    <!-- Edit mode. -->
    <template #edit>
      <OViewScope v-if="!allowRowEdit" view-mode="display" :container="'List'" :field-prefix="String(prop)">
        <div class="o-many-to-many__table" :style="{ height: tableHeightPxEdit }" tabindex="-1">
          <OVTable
            ref="ovTableRef"
            :data="tableRows"
            :row-key="'__rowKey'"
            :row-height="rowHeightEditRes"
            :header-height="headerHeight"
            :table-height="tableHeightEdit"
            :store="store"
            :base-index="1"
          >
            <OVColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'right', width: 50 }" />
            <RegistrationGuard :suppress="true">
              <slot :ref-store="relationStore" />
            </RegistrationGuard>
            <OVColumn :label="'操作'" :width="60">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="onRemove($index)"> 删除 </el-button>
              </template>
            </OVColumn>
          </OVTable>
        </div>
        <div class="o-many-to-many-actions">
          <el-button v-if="searchList" size="small" link type="primary" plain @click="openPicker"> 添加行 </el-button>
        </div>
      </OViewScope>

      <template v-else>
        <OViewScope :view-mode="binding.env.viewMode" :container="'List'" :field-prefix="String(prop)">
          <div class="o-many-to-many__table" :style="{ height: tableHeightPxEdit }" tabindex="-1">
            <OVTable
              ref="ovTableRef"
              :data="tableRows"
              :row-key="'__rowKey'"
              :row-height="rowHeightEditRes"
              :header-height="headerHeight"
              :table-height="tableHeightEdit"
              :store="store"
              :base-index="1"
            >
              <OVColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'right', width: 50 }" />
              <RegistrationGuard :suppress="true">
                <slot :ref-store="relationStore" />
              </RegistrationGuard>
              <OVColumn :label="'操作'" :width="60">
                <template #default="{ $index }">
                  <el-button link type="danger" size="small" @click="onRemove($index)"> 删除 </el-button>
                </template>
              </OVColumn>
            </OVTable>
          </div>
          <div class="o-many-to-many-actions">
            <el-button v-if="searchList" size="small" link type="primary" plain @click="openPicker"> 添加行 </el-button>
          </div>
        </OViewScope>
      </template>
    </template>

    <!-- Display mode. -->
    <template #display>
      <OViewScope view-mode="display" :container="'List'" :field-prefix="String(prop)">
        <div class="o-many-to-many__table" :style="{ height: tableHeightPxDisplay }">
          <OVTable
            :data="tableRows"
            :row-key="'__rowKey'"
            :row-height="rowHeightDisplayRes"
            :header-height="headerHeight"
            :table-height="tableHeightDisplay"
            :store="store"
            :base-index="1"
          >
            <OVColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'right', width: 50 }" />
            <RegistrationGuard :suppress="true">
              <slot :ref-store="relationStore" />
            </RegistrationGuard>
          </OVTable>
        </div>
      </OViewScope>
    </template>
  </OFieldBase>

  <el-dialog v-model="dialogVisible" :title="searchViewTitle" :width="searchViewWidth" append-to-body destroy-on-close>
    <OViewScope view-mode="display">
      <component
        v-if="searchList && relationStore"
        :is="searchList"
        ref="searchViewRef"
        :store="relationStore"
        :show-actions="false"
        :click-to-select="true"
        :height-mode="'auto'"
        :forced-condition="effectiveConditions"
        style="margin-top: -10px"
      />
    </OViewScope>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmAdd">确定</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string[]>, V = FieldPathType<T, P>">
import { computed, ref, type Component, nextTick, watch, onMounted, Ref, inject, onBeforeUnmount, defineComponent, provide } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps } from 'element-plus';
import { ElButton, ElDialog, ElMessage } from 'element-plus';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import OViewScope from '@/web/web/components/view/OViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/OListView.vue';
import { createStoreByModel } from '@/web/web/stores/registry';
import {
  exportFieldSelection,
  registerFieldPath,
  unregisterFieldPath,
  pathsToFieldSelection,
  ensureRootId,
  useFieldRegistryVersion,
} from '@/web/web/query/utils/registry/field';

defineOptions({ name: 'OManyToManyRefField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;

    showIndex?: boolean;
    rowHeightEdit?: number;
    rowHeightDisplay?: number;
    headerHeight?: number;
    minTableHeight?: number;
    maxTableHeight?: number;

    allowRowEdit?: boolean;

    searchList?: Component;
    searchViewTitle?: string;
    searchViewWidth?: string | number;

    strategy?: 'live' | 'idle' | 'blur';
    idleDelay?: number;
    commitOnBlur?: boolean;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    condition?: QueryCondition<any> | QueryCondition<any>[];
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;

    targetModel?: string;
  }>(),
  {
    label: '',
    rules: () => [],
    formItemProps: () => ({}),
    showIndex: true,
    rowHeightEdit: 60,
    rowHeightDisplay: 40,
    headerHeight: 40,
    minTableHeight: 120,
    maxTableHeight: 360,
    allowRowEdit: false,
    searchViewTitle: '选择关联项',
    searchViewWidth: '75%',
    strategy: 'live',
    idleDelay: 200,
    commitOnBlur: true,
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    condition: undefined,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

const RegistrationGuard = defineComponent({
  name: 'RegistrationGuard',
  props: {
    suppress: { type: Boolean, default: false },
  },
  setup(props, { slots }) {
    if (props.suppress) {
      provide('suppress-field-registration', true);
      // Redirect child field registration to the relation store when needed.
      provide('alt-field-registration-store', relationStore);
    }
    return () => slots.default?.();
  },
});

// Relation store for the remote model.
const relationStore = computed<WebModelStore<any> | undefined>(() => {
  if (binding.relationStore) return binding.relationStore as WebModelStore<any>;
  const target = props.targetModel || binding.meta?.relationModel;
  if (!target) return undefined;
  try {
    return createStoreByModel(target);
  } catch (e) {
    console.warn(`[OManyToManyRefField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});

const { getItems, insertItem, removeItemAt } = binding.asMutableArray();

// Cache hydrated target records to avoid duplicate fetches.
const hydratedCache = ref<Map<string, any>>(new Map());
const hydratingIds = ref<Set<string>>(new Set());
const fieldRegistryVersion = useFieldRegistryVersion();

// Row height and table height state.
const rowHeightEditRes = computed(() => props.rowHeightEdit!);
const rowHeightDisplayRes = computed(() => props.rowHeightDisplay!);
const tableHeightEdit = computed(() => {
  const body = (getItems().length || 0) * (rowHeightEditRes.value || 40);
  const total = body + (props.headerHeight || 48);
  return Math.max(props.minTableHeight!, Math.min(props.maxTableHeight!, total));
});
const tableHeightDisplay = computed(() => {
  const body = (getItems().length || 0) * (rowHeightDisplayRes.value || 40);
  const total = body + (props.headerHeight || 48);
  return Math.max(props.minTableHeight!, Math.min(props.maxTableHeight!, total));
});
const tableHeightPxEdit = computed(() => `${tableHeightEdit.value}px`);
const tableHeightPxDisplay = computed(() => `${tableHeightDisplay.value}px`);

const dialogVisible = ref(false);
const searchViewRef = ref<SelectionExpose<any> | null>(null);
const ovTableRef = ref<InstanceType<typeof OVTable> | null>(null);

// __rowKey helpers.
function readRowKeySeed(row: unknown): string | number | undefined {
  if (!row || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  return r.__rowKey ?? r.Id ?? r.id;
}
function defineHiddenRowKey(obj: any, key: string, val?: any) {
  if (!obj || typeof obj !== 'object') return;
  try {
    const hasOwn = Object.prototype.hasOwnProperty.call(obj, key);
    const enumerable = hasOwn ? Object.prototype.propertyIsEnumerable.call(obj, key) : false;
    if (!hasOwn) {
      Object.defineProperty(obj, key, {
        value: String(val ?? Math.random().toString(36).slice(2)),
        enumerable: false,
        configurable: false,
        writable: true,
      });
      return;
    }
    if (enumerable) {
      const v = val ?? obj[key];
      delete obj[key];
      Object.defineProperty(obj, key, {
        value: String(v ?? Math.random().toString(36).slice(2)),
        enumerable: false,
        configurable: false,
        writable: true,
      });
    }
  } catch {}
}
function hydrateRowKeys() {
  const arr = getItems() || [];
  for (const row of arr) {
    if (!row) continue;
    const seed = readRowKeySeed(row);
    defineHiddenRowKey(row, '__rowKey', seed);
  }
}

function extractId(v: any): string | undefined {
  if (v == null) return undefined;
  if (typeof v === 'object') return (v as any).Id ?? (v as any).id;
  return String(v);
}

// Build table rows from ids or hydrated objects while preserving row keys.
const tableRows = computed(() => {
  const items = getItems() || [];
  return items.map((item, idx) => {
    if (item && typeof item === 'object') {
      const row = item as any;
      const id = row.Id ?? row.id;
      if (id) hydratedCache.value.set(String(id), row);
      defineHiddenRowKey(row, '__rowKey', readRowKeySeed(row) ?? id ?? idx);
      return row;
    }
    const id = String(item ?? '');
    const cached = hydratedCache.value.get(id);
    if (cached) {
      defineHiddenRowKey(cached, '__rowKey', readRowKeySeed(cached) ?? id ?? idx);
      return cached;
    }
    const row = { Id: id } as any;
    defineHiddenRowKey(row, '__rowKey', id || idx);
    return row;
  });
});

// Merge onchange-driven conditions.
const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));
function toArray<T>(v: T | T[] | undefined | null): T[] {
  if (v == null) return [];
  return Array.isArray(v) ? v : [v];
}
const excludePicked = computed<QueryCondition<any> | undefined>(() => {
  const ids = (getItems() || []).map(extractId).filter(Boolean).map(String);
  if (!ids.length) return undefined;
  return ['Id', 'not in', ids] as unknown as QueryCondition<any>;
});
const externalConditions = computed<QueryCondition<any>[]>(() => toArray(props.condition));
const fieldName = computed(() => String(binding.prop));
const onchangeConditions = computed<QueryCondition<any>[]>(() => {
  const raw = lastOnchangeResult.value?.condition || [];
  return raw
    .filter((c: any) => c?.field === fieldName.value)
    .map((c: any) => c?.condition)
    .filter(Boolean);
});
const effectiveConditions = computed<QueryCondition<any> | []>(() => {
  const parts: QueryCondition<any>[] = [];
  if (excludePicked.value) parts.push(excludePicked.value);
  parts.push(...externalConditions.value, ...onchangeConditions.value);
  if (parts.length === 0) return [] as any;
  if (parts.length === 1) return parts[0];
  return { And: parts } as any;
});

// Remote field selection strategy:
// 1) Prefer non-Id fields selected on relationStore and add DisplayName as a fallback.
// 2) Otherwise fall back to parent-store selections under the current prop prefix.
// 3) Always include Id at minimum.
const remoteFields = computed<string[]>(() => {
  // depend on registry version so selection updates propagate
  void fieldRegistryVersion.value;
  const base = ['DisplayName'];

  const relStoreId = relationStore.value?.storeId;
  if (relStoreId) {
    const sel = exportFieldSelection(relStoreId) || [];
    const relNonId = sel.filter(p => p !== 'Id');
    if (relNonId.length) {
      return Array.from(new Set([...sel, ...base]));
    }
  }

  const storeId = binding.store?.storeId;
  if (storeId) {
    const prefix = `${binding.prop}.`;
    const selection = exportFieldSelection(storeId) || [];
    const picked = selection
      .filter(p => p.startsWith(prefix))
      .map(p => p.slice(prefix.length))
      .filter(Boolean);
    if (picked.length) {
      return Array.from(new Set(['Id', ...picked, ...base]));
    }
  }

  return Array.from(new Set(['Id', ...base]));
});

// Hydrate full objects by Id when the backend returns bare string ids.
function pickHydrationFields(store?: WebModelStore<any> | null) {
  const own = remoteFields.value && remoteFields.value.length ? remoteFields.value : [];
  const ensured = ensureRootId(pathsToFieldSelection(own) ?? own) || [];
  return ensured;
}

async function ensureHydrated(ids: string[]) {
  const store = relationStore.value;
  if (!store) return;
  const fields = pickHydrationFields(store);
  const missing = ids.filter(id => !hydratedCache.value.has(id) && !hydratingIds.value.has(id));
  if (!missing.length) return;

  for (const id of missing) hydratingIds.value.add(id);
  try {
    const records = await store.Search(['Id', 'in', missing] as any, { fields } as any);
    const arr = Array.isArray(records) ? records : [];
    for (const rec of arr) {
      const row = rec ? { ...(rec as any) } : null;
      if (!row) continue;
      const id = String(row.Id ?? row.id ?? '');
      defineHiddenRowKey(row, '__rowKey', readRowKeySeed(row) ?? id);
      hydratedCache.value.set(id, row);
    }
  } catch (e) {
    console.warn('[OManyToManyRefField] hydrate failed', e);
  } finally {
    for (const id of missing) hydratingIds.value.delete(id);
  }
}

watch(
  () => (getItems() || []).map(extractId).filter(Boolean).map(String),
  ids => {
    void ensureHydrated(ids);
  },
  { immediate: true }
);

// Register remote fields on relationStore so search results include them.
const registeredRefFields = ref<Set<string>>(new Set());
let registeredStoreId: string | null = null;
function syncRemoteFieldRegistration(store?: WebModelStore<any> | null, fields?: string[]) {
  const nextStoreId = store?.storeId ?? null;
  const nextSet = new Set(fields || []);
  const unchanged =
    nextStoreId === registeredStoreId && nextSet.size === registeredRefFields.value.size && Array.from(nextSet).every(f => registeredRefFields.value.has(f));
  if (unchanged) return;

  // Clean up the previous registration.
  if (registeredStoreId && registeredRefFields.value.size) {
    for (const f of registeredRefFields.value) unregisterFieldPath(registeredStoreId, f);
  }
  registeredRefFields.value.clear();
  registeredStoreId = null;

  if (!store || !nextSet.size) return;
  for (const f of nextSet) registerFieldPath(store.storeId, f);
  registeredRefFields.value = nextSet;
  registeredStoreId = store.storeId;
}

watch(
  () => [relationStore.value, remoteFields.value] as const,
  ([store, fields]) => syncRemoteFieldRegistration(store, fields),
  { immediate: true }
);

onBeforeUnmount(() => {
  syncRemoteFieldRegistration(null, []);
});

function openPicker() {
  if (!relationStore.value) {
    ElMessage.warning('[OManyToManyRefField] relationStore 未解析，无法打开选择器');
  } else {
    dialogVisible.value = true;
  }
}

function createRowKey(v: any) {
  if (!v) return v;
  const seed = readRowKeySeed(v) ?? Math.random().toString(36).slice(2);
  defineHiddenRowKey(v, '__rowKey', seed);
  return v;
}

function onRemove(i: number) {
  removeItemAt(i);
}

async function confirmAdd() {
  try {
    const expose = searchViewRef.value as any;
    const unwrap = (v: any) => (v && typeof v === 'object' && 'value' in v ? v.value : v);
    const picked = unwrap(expose?.selectedItems) as any[] | undefined;
    const toRecord = (x: any) =>
      x && typeof x === 'object' && x.kind === 'record' && x.payload ? x.payload : x && typeof x === 'object' && x.type === 'record' && x.record ? x.record : x;
    const selected: any[] = Array.isArray(picked) ? picked.map(toRecord) : [];
    const ids = selected.map(x => x?.Id ?? x?.id).filter(Boolean);
    if (!ids.length) {
      dialogVisible.value = false;
      return;
    }
    const existIds = new Set((getItems() || []).map(extractId).filter(Boolean).map(String));
    const newIds = ids.map(String).filter((id: any) => !existIds.has(id));
    if (!newIds.length) {
      dialogVisible.value = false;
      return;
    }
    const records = selected.filter(x => newIds.includes(String(x?.Id ?? x?.id)));
    for (const rec of records) insertItem(createRowKey({ ...(rec || {}) }) as any);
  } finally {
    dialogVisible.value = false;
    await nextTick();
    ovTableRef.value?.scrollToRow?.(getItems().length - 1, 'end');
  }
}

onMounted(hydrateRowKeys);
watch(
  () => getItems().length,
  () => hydrateRowKeys(),
  { immediate: true }
);

const store = props.store;

defineSlots<{
  default(props: { refStore: WebModelStore<any> | undefined }): any;
}>();
</script>

<style scoped>
.o-many-to-many__table {
  width: 100%;
  min-width: 0;
}
.o-many-to-many-actions {
  display: flex;
  align-items: center;
  padding-inline-start: 60px;
  padding-block: 6px;
}
</style>
