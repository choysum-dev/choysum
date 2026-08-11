<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue }">
      <div v-if="isTableLike" class="o-properties-summary" data-testid="o-properties-summary">
        {{ summaryText(fieldValue().value) }}
      </div>
      <div v-else class="o-properties-form" data-testid="o-properties-form">
        <div v-if="!renderableItems.length" class="o-properties-empty" data-testid="o-properties-empty" />
        <div
          v-for="item in renderableItems"
          :key="item.name"
          class="o-properties-item"
          :data-name="item.name"
          :data-type="item.type"
        >
          <label class="o-properties-item__label">{{ itemLabel(item) }}</label>
          <el-switch
            v-if="item.type === 'boolean'"
            class="o-properties-control"
            :model-value="asBoolean(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @update:model-value="(v: any) => onItemWrite(fieldValue, item.name, v)"
          />
          <el-input-number
            v-else-if="item.type === 'integer' || item.type === 'float'"
            class="o-properties-control"
            :model-value="asNumber(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            :controls="false"
            :precision="item.type === 'integer' ? 0 : undefined"
            @update:model-value="(v: any) => onItemWrite(fieldValue, item.name, v)"
          />
          <el-input
            v-else-if="item.type === 'text'"
            class="o-properties-control"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 6 }"
            :model-value="asString(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @update:model-value="(v: any) => onItemWrite(fieldValue, item.name, v)"
          />
          <el-date-picker
            v-else-if="item.type === 'date' || item.type === 'datetime'"
            class="o-properties-control"
            :type="item.type === 'datetime' ? 'datetime' : 'date'"
            :model-value="asString(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            :value-format="item.type === 'datetime' ? 'YYYY-MM-DD[T]HH:mm:ss[Z]' : 'YYYY-MM-DD[T]00:00:00[Z]'"
            @update:model-value="(v: any) => onItemWrite(fieldValue, item.name, v)"
          />
          <el-select
            v-else-if="item.type === 'selection'"
            class="o-properties-control"
            :model-value="asString(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            clearable
            @update:model-value="(v: any) => onItemWrite(fieldValue, item.name, v)"
          >
            <el-option
              v-for="opt in selectionOptions(item)"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
          <el-input
            v-else
            class="o-properties-control"
            :model-value="asString(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @update:model-value="(v: any) => onItemWrite(fieldValue, item.name, v)"
          />
        </div>
      </div>
    </template>

    <template #display="{ fieldValue }">
      <span v-if="isTableLike" class="o-properties-summary" data-testid="o-properties-summary">
        {{ summaryText(fieldValue().value) }}
      </span>
      <div v-else class="o-properties-form o-properties-form--display" data-testid="o-properties-form">
        <div v-if="!renderableItems.length" class="o-properties-empty" data-testid="o-properties-empty" />
        <div
          v-for="item in renderableItems"
          :key="item.name"
          class="o-properties-item o-properties-item--display"
          :data-name="item.name"
        >
          <span class="o-properties-item__label">{{ itemLabel(item) }}</span>
          <span class="o-properties-item__value">{{ displayItemValue(fieldValue().value, item) }}</span>
        </div>
      </div>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, Record<string, any> | null | undefined>, V = FieldPathType<T, P>">
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RuleItem } from 'async-validator';
import {
  ElInput,
  ElInputNumber,
  ElSwitch,
  ElSelect,
  ElOption,
  ElDatePicker,
  type FormItemProps,
} from 'element-plus';
import { computed, ref, watch, type WritableComputedRef } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import {
  countSchemaMapIntersection,
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  writePropertyValue,
  type PropertiesMap,
} from './oproperties_helpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/OPropertiesField' });

defineOptions({ name: 'OPropertiesField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    /** Optional unsaved parent container id override (PP4 B1 re-resolve). */
    containerId?: string | null;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: {
      width?: number | string;
      minWidth?: number;
      align?: 'left' | 'center' | 'right';
      fixed?: 'left' | 'right';
      sortable?: boolean;
    };
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

const isTableLike = computed(() => {
  const mode = props.renderMode;
  if (mode === 'table' || mode === 'inline') return true;
  if (mode === 'form') return false;
  return binding.env?.isForm === false;
});

const toView = (raw: any): PropertiesMap => {
  if (raw == null) return {};
  if (raw && typeof raw === 'object' && !Array.isArray(raw)) return { ...(raw as PropertiesMap) };
  return {};
};
const fromView = (v: PropertiesMap) => v as unknown as V;

const resolvedItems = ref<ResolvedPropertyItem[]>([]);
const schemaNames = computed(() => resolvedItems.value.map(i => i.name).filter(Boolean));
const renderableItems = computed(() => filterRenderablePropertyItems(resolvedItems.value).renderable);

function itemLabel(item: ResolvedPropertyItem): string {
  return String(item.string || item.name);
}

function selectionOptions(item: ResolvedPropertyItem) {
  return normalizeSelectionOptions(item.selection);
}

function itemValue(map: unknown, item: ResolvedPropertyItem): unknown {
  const m = toView(map);
  if (Object.prototype.hasOwnProperty.call(m, item.name)) return m[item.name];
  if (Object.prototype.hasOwnProperty.call(item, 'value')) return item.value;
  if (Object.prototype.hasOwnProperty.call(item, 'default')) return item.default;
  return undefined;
}

function asBoolean(v: unknown): boolean {
  return v === true;
}
function asNumber(v: unknown): number | undefined {
  return typeof v === 'number' && Number.isFinite(v) ? v : undefined;
}
function asString(v: unknown): string {
  if (v == null) return '';
  return String(v);
}

function displayItemValue(map: unknown, item: ResolvedPropertyItem): string {
  const v = itemValue(map, item);
  if (v == null) return '';
  if (item.type === 'boolean') return v === true ? _t('Yes') : _t('No');
  if (item.type === 'selection') {
    const opt = selectionOptions(item).find(o => o.value === v);
    return opt?.label ?? String(v);
  }
  return String(v);
}

function summaryText(map: unknown): string {
  const n = countSchemaMapIntersection(schemaNames.value, map);
  return n > 0 ? _t('%s properties', n) : '';
}

function onItemWrite(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  value: unknown
) {
  const cur = fieldValue().value;
  fieldValue().value = writePropertyValue(resolvedItems.value, cur, name, value);
}

async function reloadResolved() {
  const store = (binding.store ?? props.store) as WebModelStore<T> | undefined;
  const fieldName = String(binding.prop || props.prop || '');
  if (!store || !fieldName || typeof (store as any).ResolveProperties !== 'function') {
    resolvedItems.value = [];
    return;
  }
  const record = binding.recordRef?.()?.value ?? {};
  const map = toView(binding.fieldRef?.()?.value);
  const payload = { ...(record as any), [fieldName]: map };
  const opts =
    props.containerId !== undefined ? { containerId: props.containerId } : undefined;
  try {
    const items = await (store as any).ResolveProperties(payload, fieldName, opts);
    const list = Array.isArray(items) ? (items as ResolvedPropertyItem[]) : [];
    const { skipped } = filterRenderablePropertyItems(list);
    for (const item of skipped) {
      // Historical dirty Definition types: skip without breaking the form.
      console.warn(`[OPropertiesField] skipping unsupported property type '${item.type}' (${item.name})`);
    }
    resolvedItems.value = list;
  } catch (e) {
    console.warn('[OPropertiesField] ResolveProperties failed', e);
    resolvedItems.value = [];
  }
}

watch(
  () => [
    binding.recordRef?.()?.value,
    binding.fieldRef?.()?.value,
    props.containerId,
    binding.prop,
    props.store,
    isTableLike.value,
  ],
  () => {
    void reloadResolved();
  },
  { deep: true, immediate: true }
);
</script>

<style scoped lang="scss">
.o-properties-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.o-properties-item {
  display: grid;
  grid-template-columns: minmax(96px, 28%) 1fr;
  gap: 8px;
  align-items: center;
}
.o-properties-item__label {
  color: var(--el-text-color-regular);
  font-size: 13px;
}
.o-properties-control {
  width: 100%;
}
.o-properties-summary {
  font-size: 13px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.o-properties-empty {
  min-height: 4px;
}
</style>
