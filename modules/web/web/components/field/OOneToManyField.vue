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
    <template #edit>
      <OViewScope :view-mode="binding.env.viewMode" :container="'List'" :field-prefix="String(prop)">
        <div class="o-one-to-many__table" :style="{ height: tableHeightPxEdit }" tabindex="-1">
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
        <div class="o-one-to-many-actions">
          <el-button size="small" link type="primary" plain @click="handleAddItem">{{ _t('Add row') }}</el-button>
        </div>
      </OViewScope>
    </template>
    <template #display>
      <OViewScope view-mode="display" :container="'List'" :field-prefix="String(prop)">
        <div class="o-one-to-many__table" :style="{ height: tableHeightPxDisplay }">
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
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, nextTick, ref, watch, onMounted } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps } from 'element-plus';
import { ElButton } from 'element-plus';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import OVTable from '@/web/web/components/vtable/OVTable.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import OViewScope from '@/web/web/components/view/OViewScope.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OOneToManyField' });

defineOptions({ name: 'OOneToManyField', inheritAttrs: false });

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
    defaultRecord?: Record<string, any> | (() => Record<string, any>);

    rowHeightEdit?: number; // Row height in edit mode.
    rowHeightDisplay?: number; // Row height in display mode.

    headerHeight?: number;
    minTableHeight?: number;
    maxTableHeight?: number;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    strategy?: 'live' | 'idle' | 'blur';
    idleDelay?: number;
    commitOnBlur?: boolean;
    allowRowEdit?: boolean;
    // Added render mode and inline error support.
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    label: '',
    rules: () => [],
    formItemProps: () => ({}),
    showIndex: true,

    rowHeightEdit: 60,
    rowHeightDisplay: 40,

    defaultRecord: () => ({}),
    minTableHeight: 120,
    maxTableHeight: 360,
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    strategy: 'live',
    idleDelay: 200,
    commitOnBlur: true,
    allowRowEdit: false,
    renderMode: 'auto',
    showInlineError: false,
  }
);

// Field binding.
const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;
binding.registerFields(`${binding.prop}.DisplayName`);

const { getItems, insertItem, removeItemAt } = binding.asMutableArray<any>();
const store = props.store;

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
const ovTableRef = ref<InstanceType<typeof OVTable> | null>(null);

// Read the row-key seed without relying on generic field properties.
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

// Ensure the collection exists before adding rows on new records.
function ensureArrayInitialized() {
  const refVal = binding.fieldRef() as any;
  if (!Array.isArray(refVal.value)) {
    refVal.value = [];
  }
}

// Remove a row.
function onRemove(i: number) {
  removeItemAt(i);
}

// Add a row.
async function handleAddItem() {
  ensureArrayInitialized();
  const targetIndex = getItems().length;
  insertItem(makeDefaultItem());
  await nextTick();
  ovTableRef.value?.scrollToRow?.(targetIndex, 'end');
}

// Build the default new row.
function makeDefaultItem(): Record<string, any> {
  const rec = typeof props.defaultRecord === 'function' ? (props.defaultRecord as any)() : props.defaultRecord;
  const row = { ...(rec || {}) };
  const seed = readRowKeySeed(row) ?? Math.random().toString(36).slice(2);
  defineHiddenRowKey(row, '__rowKey', seed);
  return row;
}

// Keep row keys hydrated.
onMounted(hydrateRowKeys);
watch(
  () => getItems().length,
  () => hydrateRowKeys(),
  { immediate: true }
);
</script>

<style scoped>
.o-one-to-many__table {
  width: 100%;
  min-width: 0;
}
.o-one-to-many-actions {
  display: flex;
  align-items: center;
  padding-inline-start: 60px;
  padding-block: 6px;
}
</style>
