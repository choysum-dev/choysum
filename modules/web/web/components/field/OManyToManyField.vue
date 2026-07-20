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
            :data="getItems()"
            :row-key="'__rowKey'"
            :row-height="rowHeightEditRes"
            :header-height="headerHeight"
            :table-height="tableHeightEdit"
            :store="store"
            :base-index="1"
          >
            <OVColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'right', width: 50 }" />
            <slot />
            <OVColumn :label="_t('Actions')" :width="60">
              <template #default="{ $index }">
                <el-button link type="danger" size="small" @click="onRemove($index)">{{ _t('Delete') }}</el-button>
              </template>
            </OVColumn>
          </OVTable>
        </div>
        <div class="o-many-to-many-actions">
          <el-button v-if="searchList" size="small" link type="primary" plain @click="openPicker">{{ _t('Add row') }}</el-button>
        </div>
      </OViewScope>

      <template v-else>
        <OViewScope :view-mode="binding.env.viewMode" :container="'List'" :field-prefix="String(prop)">
          <div class="o-many-to-many__table" :style="{ height: tableHeightPxEdit }" tabindex="-1">
            <OVTable
              ref="ovTableRef"
              :data="getItems()"
              :row-key="'__rowKey'"
              :row-height="rowHeightEditRes"
              :header-height="headerHeight"
              :table-height="tableHeightEdit"
              :store="store"
              :base-index="1"
            >
              <OVColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'right', width: 50 }" />
              <slot />
              <OVColumn :label="_t('Actions')" :width="60">
                <template #default="{ $index }">
                  <el-button link type="danger" size="small" @click="onRemove($index)">{{ _t('Delete') }}</el-button>
                </template>
              </OVColumn>
            </OVTable>
          </div>
          <div class="o-many-to-many-actions">
            <el-button v-if="searchList" size="small" link type="primary" plain @click="openPicker">{{ _t('Add row') }}</el-button>
          </div>
        </OViewScope>
      </template>
    </template>

    <!-- Display mode. -->
    <template #display>
      <OViewScope view-mode="display" :container="'List'" :field-prefix="String(prop)">
        <div class="o-many-to-many__table" :style="{ height: tableHeightPxDisplay }">
          <OVTable
            :data="getItems()"
            :row-key="'__rowKey'"
            :row-height="rowHeightDisplayRes"
            :header-height="headerHeight"
            :table-height="tableHeightDisplay"
            :store="store"
            :base-index="1"
          >
            <OVColumn v-if="showIndex" type="index" label="#" :vColumnProps="{ align: 'right', width: 50 }" />
            <slot />
          </OVTable>
        </div>
      </OViewScope>
    </template>
  </OFieldBase>

  <el-dialog v-model="dialogVisible" :title="effectiveSearchViewTitle" :width="searchViewWidth" append-to-body destroy-on-close>
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
        <el-button @click="dialogVisible = false">{{ _t('Cancel') }}</el-button>
        <el-button type="primary" @click="confirmAdd">{{ _t('OK') }}</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, type Component, nextTick, watch, onMounted, onBeforeUnmount, inject, Ref } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel, QueryCondition } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps } from 'element-plus';
import { ElButton, ElDialog, ElMessage } from 'element-plus';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import OViewScope from '@/web/web/components/view/OViewScope.vue';
import type { SelectionExpose } from '@/web/web/components/view/listViewTypes';
import { createStoreByModel } from '@/web/web/stores/registry';
import { useProvidedOnchange } from '@/web/web/composables/useOnchange';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OManyToManyField' });

defineOptions({ name: 'OManyToManyField', inheritAttrs: false });

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
    rowHeightEdit?: number; // Row height in edit mode.
    rowHeightDisplay?: number; // Row height in display mode.
    headerHeight?: number;
    minTableHeight?: number;
    maxTableHeight?: number;

    allowRowEdit?: boolean;

    searchList?: Component;
    searchViewTitle?: string;
    searchViewWidth?: string | number;
    targetModel?: string;

    strategy?: 'live' | 'idle' | 'blur';
    idleDelay?: number;
    commitOnBlur?: boolean;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    /* External static constraints merged with onchange conditions. */
    condition?: QueryCondition<any> | QueryCondition<any>[];
    // Added render mode and inline error support.
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    showIndex: true,
    rowHeightEdit: 60,
    rowHeightDisplay: 40,
    headerHeight: 40,
    minTableHeight: 120,
    maxTableHeight: 360,
    allowRowEdit: false,
    searchViewTitle: '',
    searchViewWidth: '75%',
    targetModel: '',

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

const effectiveSearchViewTitle = computed(() => props.searchViewTitle || _t('Select related items'));

// Field binding.
const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;
binding.registerFields(`${binding.prop}.DisplayName`);

const relationStore = computed<WebModelStore<any> | undefined>(() => {
  if (binding.relationStore) return binding.relationStore as WebModelStore<any>;
  const target = props.targetModel || binding.meta?.relationModel;
  if (!target) return undefined;
  try {
    return createStoreByModel(target);
  } catch (e) {
    console.warn(`[OManyToManyField] Failed to create store for model '${target}'`, e);
    return undefined;
  }
});
const { getItems, insertItem, removeItemAt } = binding.asMutableArray();

// Row height for edit and display modes.
const rowHeightEditRes = computed(() => props.rowHeightEdit!);
const rowHeightDisplayRes = computed(() => props.rowHeightDisplay!);

// Table height for edit and display modes.
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

// Read the row-key seed.
function readRowKeySeed(row: unknown): string | number | undefined {
  if (!row || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  return r.__rowKey ?? r.Id ?? r.id;
}

// Define a non-enumerable __rowKey.
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

// Ensure existing and future rows all have a hidden __rowKey.
function hydrateRowKeys() {
  const arr = getItems() || [];
  for (const row of arr) {
    if (!row) continue;
    const seed = readRowKeySeed(row);
    defineHiddenRowKey(row, '__rowKey', seed);
  }
}

// Use the cumulative onchange result injected by OFormView.
const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));

// Utility helper.
function toArray<T>(v: T | T[] | undefined | null): T[] {
  if (v == null) return [];
  return Array.isArray(v) ? v : [v];
}

// Condition: exclude already selected rows.
const excludePicked = computed<QueryCondition<any> | undefined>(() => {
  const ids = (getItems() || []).map((x: any) => x?.Id).filter(Boolean);
  if (!ids.length) return undefined;
  return ['Id', 'not in', ids] as unknown as QueryCondition<any>;
});

// Condition: externally provided props.condition.
const externalConditions = computed<QueryCondition<any>[]>(() => toArray(props.condition));

// Condition: onchange output for the current field.
const fieldName = computed(() => String(binding.prop));
const onchangeConditions = computed<QueryCondition<any>[]>(() => {
  const raw = lastOnchangeResult.value?.condition || [];
  return raw
    .filter((c: any) => c?.field === fieldName.value)
    .map((c: any) => c?.condition)
    .filter(Boolean);
});

// Effective conditions: exclude picked rows, then merge external and onchange filters.
// When no condition exists, return [] explicitly so downstream consumers clear overrides.
const effectiveConditions = computed<QueryCondition<any> | []>(() => {
  const parts: QueryCondition<any>[] = [];
  if (excludePicked.value) parts.push(excludePicked.value);
  parts.push(...externalConditions.value, ...onchangeConditions.value);

  if (parts.length === 0) return [] as any;
  if (parts.length === 1) return parts[0];
  return { And: parts } as any;
});

// Picker actions.
function openPicker() {
  if (!relationStore.value) {
    ElMessage.warning(_t('relationStore is unresolved; cannot open picker'));
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
    // Unwrap selectedItems regardless of whether it is exposed as a ref or computed.
    const expose = searchViewRef.value as any;
    const unwrap = (v: any) => (v && typeof v === 'object' && 'value' in v ? v.value : v);
    const picked = unwrap(expose?.selectedItems) as any[] | undefined;

    // Normalize wrapped row payloads exposed by OListView.
    const toRecord = (x: any) =>
      x && typeof x === 'object' && x.kind === 'record' && x.payload ? x.payload : x && typeof x === 'object' && x.type === 'record' && x.record ? x.record : x;
    const selected: any[] = Array.isArray(picked) ? picked.map(toRecord) : [];
    const ids = selected.map(x => x?.Id ?? x?.id).filter(Boolean);
    if (!ids.length) {
      dialogVisible.value = false;
      return;
    }
    const existIds = new Set((getItems() || []).map((x: any) => x?.Id).filter(Boolean));
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

// Keep row keys hydrated.
onMounted(hydrateRowKeys);
watch(
  () => getItems().length,
  () => hydrateRowKeys(),
  { immediate: true }
);

// Expose store for the template.
const store = props.store;
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
