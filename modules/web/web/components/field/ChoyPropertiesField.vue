<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FieldBase
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
      <div v-if="isTableLike" class="truncate text-[13px] text-foreground" data-testid="choy-properties-summary">
        {{ summaryText(fieldValue().value) }}
      </div>
      <div v-else class="flex w-full flex-col gap-2" data-testid="choy-properties-form">
        <div v-if="!renderableItems.length" class="min-h-1" data-testid="choy-properties-empty" />
        <div
          v-for="item in renderableItems"
          :key="item.name"
          class="grid grid-cols-[minmax(96px,28%)_1fr] items-center gap-2"
          :data-name="item.name"
          :data-type="item.type"
          :data-readonly="item.readonly ? '1' : '0'"
        >
          <label class="text-[13px] text-foreground" :for="controlId(item)">{{ itemLabel(item) }}</label>
          <input
            type="checkbox"
            v-if="item.type === 'boolean'"
            class="w-full"
            :id="controlId(item)"
            :checked="asBoolean(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @change="onItemWrite(fieldValue, item.name, ($event.target as HTMLInputElement).checked)"
          />
          <input
            v-else-if="item.type === 'integer' || item.type === 'float'"
            class="w-full"
            :id="controlId(item)"
            type="number"
            :value="asNumber(itemValue(fieldValue().value, item)) ?? ''"
            :disabled="!!item.readonly"
            :step="item.type === 'integer' ? 1 : 'any'"
            @change="onItemWrite(fieldValue, item.name, ($event.target as HTMLInputElement).value === '' ? null : Number(($event.target as HTMLInputElement).value))"
          />
          <textarea
            v-else-if="item.type === 'text'"
            class="w-full"
            :id="controlId(item)"
            rows="3"
            :value="asString(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @input="onItemWrite(fieldValue, item.name, ($event.target as HTMLTextAreaElement).value)"
          ></textarea>
          <input
            v-else-if="item.type === 'date'"
            class="w-full"
            :id="controlId(item)"
            type="date"
            :value="dateInputValue(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @change="onDateWrite(fieldValue, item.name, ($event.target as HTMLInputElement).value)"
          />
          <input
            v-else-if="item.type === 'datetime'"
            class="w-full"
            :id="controlId(item)"
            type="datetime-local"
            :value="datetimePickerValue(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @change="onDatetimeWrite(fieldValue, item.name, ($event.target as HTMLInputElement).value)"
          />
          <select
            v-else-if="item.type === 'selection'"
            class="w-full"
            :id="controlId(item)"
            :value="asString(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @change="onItemWrite(fieldValue, item.name, ($event.target as HTMLSelectElement).value)"
          >
            <option
              v-for="opt in selectionOptions(item)"
              :key="opt.value"
              :value="opt.value"
            >
              {{ opt.label }}
            </option>
          </select>
          <input
            v-else
            class="w-full"
            :id="controlId(item)"
            :value="asString(itemValue(fieldValue().value, item))"
            :disabled="!!item.readonly"
            @input="onItemWrite(fieldValue, item.name, ($event.target as HTMLInputElement).value)"
          />
        </div>
      </div>
    </template>

    <template #display="{ fieldValue }">
      <span v-if="isTableLike" class="truncate text-[13px] text-foreground" data-testid="choy-properties-summary">
        {{ summaryText(fieldValue().value) }}
      </span>
      <div v-else class="choy-properties-form choy-properties-form--display flex w-full flex-col gap-2" data-testid="choy-properties-form">
        <div v-if="!renderableItems.length" class="min-h-1" data-testid="choy-properties-empty" />
        <div
          v-for="item in renderableItems"
          :key="item.name"
          class="choy-properties-item choy-properties-item--display grid grid-cols-[minmax(96px,28%)_1fr] items-center gap-2"
          :data-name="item.name"
        >
          <span class="text-[13px] text-foreground">{{ itemLabel(item) }}</span>
          <span class="choy-properties-item__value">{{ displayItemValue(fieldValue().value, item) }}</span>
        </div>
      </div>
    </template>
  </FieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, Record<string, any> | null | undefined>, V = FieldPathType<T, P>">
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RuleItem } from 'async-validator';
import { computed, ref, watch, type WritableComputedRef } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import FieldBase, { type FieldStateExpr, type FormItemProps } from './FieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import {
  countSchemaMapIntersection,
  filterRenderablePropertyItems,
  normalizeSelectionOptions,
  propertiesFieldKey,
  propertyDateFromInput,
  propertyDateToInput,
  propertyDatetimeFromPicker,
  propertyDatetimeToPicker,
  writePropertyValue,
  type PropertiesMap
} from './propertiesHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/PropertiesField' });

defineOptions({ name: 'ChoyPropertiesField' });

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
  if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) return Object.create(null);
  // Copy own keys onto a null-prototype object so names like "__proto__" survive.
  const out: PropertiesMap = Object.create(null);
  for (const key of Object.keys(raw as object)) {
    out[key] = (raw as Record<string, unknown>)[key];
  }
  return out;
};
const fromView = (v: PropertiesMap) => v as unknown as V;

const resolvedItems = ref<ResolvedPropertyItem[]>([]);
const schemaNames = computed(() => resolvedItems.value.map(i => i.name).filter(Boolean));
const renderableItems = computed(() => filterRenderablePropertyItems(resolvedItems.value).renderable);
let resolveGeneration = 0;

function itemLabel(item: ResolvedPropertyItem): string {
  return String(item.string || item.name);
}

function controlId(item: ResolvedPropertyItem): string {
  return `choy-properties-${propertiesFieldKey(binding.prop, props.prop)}-${item.name}`;
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

function dateInputValue(raw: unknown): string {
  return propertyDateToInput(raw);
}

function datetimePickerValue(raw: unknown): Date | null {
  return propertyDatetimeToPicker(raw);
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

function onDateWrite(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  value: unknown
) {
  onItemWrite(fieldValue, name, propertyDateFromInput(value));
}

function onDatetimeWrite(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  value: unknown
) {
  onItemWrite(fieldValue, name, propertyDatetimeFromPicker(value));
}

async function reloadResolved() {
  const generation = ++resolveGeneration;
  const store = (binding.store ?? props.store) as WebModelStore<T> | undefined;
  const fieldName = propertiesFieldKey(binding.prop, props.prop, '');
  if (!store || !fieldName || typeof (store as any).ResolveProperties !== 'function') {
    // Sync bail-out: generation cannot advance between ++ and here.
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
    if (generation !== resolveGeneration) return;
    const list = Array.isArray(items) ? (items as ResolvedPropertyItem[]) : [];
    const { skipped } = filterRenderablePropertyItems(list);
    for (const item of skipped) {
      // Historical dirty Definition types: skip without breaking the form.
      console.warn(`[OPropertiesField] skipping unsupported property type '${item.type}' (${item.name})`);
    }
    resolvedItems.value = list;
  } catch (e) {
    if (generation !== resolveGeneration) return;
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

