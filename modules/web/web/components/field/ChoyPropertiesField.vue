<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :orientation="orientation"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    :label-for-control="false"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue, inputId, labelId, describedBy }">
      <div v-if="isTableLike" class="truncate text-[13px] text-foreground" data-testid="choy-properties-summary">
        {{ summaryText(fieldValue().value) }}
      </div>
      <div
        v-else
        class="flex w-full flex-col gap-1.5"
        data-testid="choy-properties-form"
        :id="inputId || undefined"
        :aria-labelledby="labelId || undefined"
        :aria-describedby="describedBy || undefined"
      >
        <slot v-if="!renderableItems.length" name="empty">
          <p class="m-0 text-sm text-muted-foreground" data-testid="choy-properties-empty">
            {{ _t('No properties defined') }}
          </p>
        </slot>
        <template v-for="item in renderableItems" :key="item.name">
          <dl
            v-if="item.readonly"
            class="m-0 grid grid-cols-[minmax(4.5rem,7.5rem)_minmax(0,1fr)] items-center gap-x-2 gap-y-1"
            :data-name="item.name"
            :data-type="item.type"
            data-readonly="1"
          >
            <dt class="text-[13px] font-normal text-muted-foreground">{{ itemLabel(item) }}</dt>
            <dd
              class="m-0 truncate text-sm text-muted-foreground"
              data-testid="choy-properties-readonly-value"
            >{{ displayItemValue(fieldValue().value, item) }}</dd>
          </dl>
          <div
            v-else
            class="grid grid-cols-[minmax(4.5rem,7.5rem)_minmax(0,1fr)] items-center gap-x-2 gap-y-1"
            :data-name="item.name"
            :data-type="item.type"
            data-readonly="0"
          >
            <label class="text-[13px] text-foreground" :for="controlId(item)">{{ itemLabel(item) }}</label>
            <Switch
              v-if="item.type === 'boolean'"
              class="justify-self-start"
              :id="controlId(item)"
              :model-value="asBoolean(itemValue(fieldValue().value, item))"
              @update:model-value="onItemWrite(fieldValue, item.name, $event === true)"
            />
            <Input
              v-else-if="item.type === 'integer' || item.type === 'float'"
              :ref="(el) => setNumberInputHost(item.name, el)"
              class="w-full text-end"
              :id="controlId(item)"
              type="text"
              :inputmode="item.type === 'integer' ? 'numeric' : 'decimal'"
              :model-value="numberDraftValue(item, itemValue(fieldValue().value, item))"
              @update:model-value="onNumberInput(fieldValue, item.name, item.type, $event)"
              @blur="onNumberBlur(fieldValue, item.name, item.type)"
            />
            <Textarea
              v-else-if="item.type === 'text'"
              class="w-full min-h-[4.5em]"
              :id="controlId(item)"
              :rows="3"
              :model-value="asString(itemValue(fieldValue().value, item))"
              @update:model-value="onItemWrite(fieldValue, item.name, $event)"
            />
            <DatePicker
              v-else-if="item.type === 'date'"
              class="w-full"
              :id="controlId(item)"
              :model-value="datePickerModel(itemValue(fieldValue().value, item))"
              @update:model-value="onDateWrite(fieldValue, item.name, $event)"
            />
            <DateTimePicker
              v-else-if="item.type === 'datetime'"
              class="w-full"
              :id="controlId(item)"
              :model-value="datetimePickerModel(itemValue(fieldValue().value, item))"
              @update:model-value="onDatetimeWrite(fieldValue, item.name, $event)"
            />
            <Select
              v-else-if="item.type === 'selection'"
              :model-value="selectionModel(itemValue(fieldValue().value, item))"
              @update:model-value="onItemWrite(fieldValue, item.name, $event ?? '')"
            >
              <SelectTrigger class="w-full" :id="controlId(item)" />
              <SelectContent>
                <SelectItem
                  v-for="opt in selectionOptions(item)"
                  :key="opt.value"
                  :value="opt.value"
                >
                  {{ translateText(opt.label) }}
                </SelectItem>
              </SelectContent>
            </Select>
            <Input
              v-else
              class="w-full"
              :id="controlId(item)"
              :model-value="asString(itemValue(fieldValue().value, item))"
              @update:model-value="onItemWrite(fieldValue, item.name, $event)"
            />
          </div>
        </template>
      </div>
    </template>

    <template #display="{ fieldValue }">
      <span v-if="isTableLike" class="truncate text-[13px] text-foreground" data-testid="choy-properties-summary">
        {{ summaryText(fieldValue().value) }}
      </span>
      <div
        v-else
        class="choy-properties-form choy-properties-form--display flex w-full flex-col gap-1.5"
        data-testid="choy-properties-form"
      >
        <slot v-if="!renderableItems.length" name="empty">
          <p class="m-0 text-sm text-muted-foreground" data-testid="choy-properties-empty">
            {{ _t('No properties defined') }}
          </p>
        </slot>
        <div
          v-for="item in renderableItems"
          :key="item.name"
          class="choy-properties-item choy-properties-item--display grid grid-cols-[minmax(4.5rem,7.5rem)_minmax(0,1fr)] items-center gap-x-2 gap-y-1"
          :data-name="item.name"
        >
          <span class="text-[13px] text-muted-foreground">{{ itemLabel(item) }}</span>
          <span class="choy-properties-item__value truncate text-sm text-foreground">{{
            displayItemValue(fieldValue().value, item)
          }}</span>
        </div>
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, Record<string, any> | null | undefined>, V = FieldPathType<T, P>">
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RuleItem } from 'async-validator';
import { computed, ref, watch, type WritableComputedRef } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import Input from '../vendor/ui/input/Input.vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import Switch from '../vendor/ui/switch/Switch.vue';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import DatePicker from '../internal/DatePicker.vue';
import DateTimePicker from '../internal/DateTimePicker.vue';
import { resolveNumericInputEl, resyncControlledNumericRaw } from './numericFieldHelpers';
import {
  filterRenderablePropertyItems,
  formatPropertiesSummary,
  isPropertyFloatIntermediate,
  isPropertyFloatRawAllowed,
  isPropertyIntegerIntermediate,
  isPropertyIntegerRawAllowed,
  normalizeSelectionOptions,
  propertiesFieldKey,
  propertiesRecordWatchFingerprint,
  propertyDateFromInput,
  propertyDateToDisplay,
  propertyDateToInput,
  propertyDatetimeFromPickerModel,
  propertyDatetimeToDisplay,
  propertyDatetimeToPickerModel,
  propertyFloatFromInput,
  propertyIntegerFromInput,
  writePropertyValue,
  type PropertiesMap,
} from './propertiesHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyPropertiesField' });

defineOptions({ name: 'ChoyPropertiesField' });

defineSlots<{
  /** Optional empty-state chrome (e.g. link to a definition editor). */
  empty?: () => unknown;
}>();

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
    orientation?: 'horizontal' | 'vertical';
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
  },
);

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

const isTableLike = computed(() => {
  const mode = props.renderMode;
  if (mode === 'table' || mode === 'inline') return true;
  if (mode === 'form') return false;
  return binding.env?.isForm === false;
});

const fieldName = computed(() => propertiesFieldKey(binding.prop, props.prop, ''));

const toView = (raw: any): PropertiesMap => {
  if (raw == null || typeof raw !== 'object' || Array.isArray(raw)) return Object.create(null);
  const out: PropertiesMap = Object.create(null);
  for (const key of Object.keys(raw as object)) {
    out[key] = (raw as Record<string, unknown>)[key];
  }
  return out;
};
const fromView = (v: PropertiesMap) => v as unknown as V;

const resolvedItems = ref<ResolvedPropertyItem[]>([]);
const renderableItems = computed(() => filterRenderablePropertyItems(resolvedItems.value).renderable);
/** In-progress integer/float keystrokes (preserves intermediate `-` / `3.` like ChoyInt/NumberField). */
const numberDrafts = ref<Record<string, string>>(Object.create(null));
/** Vue Input hosts for numeric rows — used to resync DOM when rejecting letters. */
const numberInputHosts = new Map<string, unknown>();
let resolveGeneration = 0;

function setNumberInputHost(name: string, el: unknown) {
  if (el) numberInputHosts.set(name, el);
  else numberInputHosts.delete(name);
}

/** Schema / selection labels are msgid-shaped; missing catalog entries fall back to the source string. */
function translateText(text: unknown): string {
  const s = String(text ?? '').trim();
  return s ? _t(s) : '';
}

function itemLabel(item: ResolvedPropertyItem): string {
  return translateText(item.string || item.name) || String(item.name);
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

function numberDraftValue(item: ResolvedPropertyItem, raw: unknown): string {
  if (Object.prototype.hasOwnProperty.call(numberDrafts.value, item.name)) {
    return numberDrafts.value[item.name]!;
  }
  const n = asNumber(raw);
  return n == null ? '' : String(n);
}

function clearNumberDraft(name: string) {
  if (!Object.prototype.hasOwnProperty.call(numberDrafts.value, name)) return;
  const next = { ...numberDrafts.value };
  delete next[name];
  numberDrafts.value = next;
}

/** Keep browser caret value in sync when a keystroke is rejected (same as ChoyIntField). */
function resyncNumberInput(
  name: string,
  fieldValue: () => WritableComputedRef<any> | { value: any },
) {
  let keep = '';
  if (Object.prototype.hasOwnProperty.call(numberDrafts.value, name)) {
    keep = numberDrafts.value[name]!;
  } else {
    const n = asNumber(toView(fieldValue().value)[name]);
    keep = n == null ? '' : String(n);
  }
  const editingRaw = ref<string | null>(keep);
  const byRef = resolveNumericInputEl(numberInputHosts.get(name));
  const byId =
    typeof document !== 'undefined'
      ? (document.getElementById(
          `choy-properties-${propertiesFieldKey(binding.prop, props.prop)}-${name}`,
        ) as HTMLInputElement | null)
      : null;
  resyncControlledNumericRaw(editingRaw, byRef || byId);
}

function datePickerModel(raw: unknown): string | null {
  const s = propertyDateToInput(raw);
  return s || null;
}

function datetimePickerModel(raw: unknown): string | null {
  return propertyDatetimeToPickerModel(raw);
}

function selectionModel(raw: unknown): string | null {
  if (raw == null || raw === '') return null;
  return String(raw);
}

function displayItemValue(map: unknown, item: ResolvedPropertyItem): string {
  const v = itemValue(map, item);
  if (v == null) return '';
  if (item.type === 'boolean') return v === true ? _t('Yes') : _t('No');
  if (item.type === 'selection') {
    const opt = selectionOptions(item).find(o => o.value === v);
    return translateText(opt?.label ?? v);
  }
  if (item.type === 'date') return propertyDateToDisplay(v);
  if (item.type === 'datetime') return propertyDatetimeToDisplay(v);
  return String(v);
}

function summaryText(map: unknown): string {
  return formatPropertiesSummary({
    items: renderableItems.value,
    map,
    formatValue: (item, raw) => displayItemValue({ [item.name]: raw }, item),
    formatPair: (item, displayValue) => `${itemLabel(item)}: ${displayValue}`,
    moreLabel: remaining => _t('+%s more', remaining),
    maxItems: 2,
  });
}

function onItemWrite(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  value: unknown,
) {
  const cur = fieldValue().value;
  fieldValue().value = writePropertyValue(resolvedItems.value, cur, name, value);
}

function onNumberInput(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  type: string,
  raw: string,
) {
  if (type === 'integer') {
    if (!isPropertyIntegerRawAllowed(raw)) {
      resyncNumberInput(name, fieldValue);
      return;
    }
    numberDrafts.value = { ...numberDrafts.value, [name]: raw };
    if (isPropertyIntegerIntermediate(raw)) return;
    const n = propertyIntegerFromInput(raw);
    if (n == null) return;
    onItemWrite(fieldValue, name, n);
    return;
  }
  if (!isPropertyFloatRawAllowed(raw)) {
    resyncNumberInput(name, fieldValue);
    return;
  }
  numberDrafts.value = { ...numberDrafts.value, [name]: raw };
  if (isPropertyFloatIntermediate(raw)) return;
  const n = propertyFloatFromInput(raw);
  if (n == null) return;
  onItemWrite(fieldValue, name, n);
}

function onNumberBlur(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  type: string,
) {
  const raw = Object.prototype.hasOwnProperty.call(numberDrafts.value, name)
    ? numberDrafts.value[name]!
    : null;
  if (raw == null) return;
  if (type === 'integer') {
    if (raw === '' || raw === '-') {
      onItemWrite(fieldValue, name, null);
    } else {
      const n = propertyIntegerFromInput(raw);
      if (n != null) onItemWrite(fieldValue, name, n);
    }
    clearNumberDraft(name);
    return;
  }
  if (raw === '' || isPropertyFloatIntermediate(raw)) {
    const final = propertyFloatFromInput(raw.replace(/\.$/, ''));
    onItemWrite(fieldValue, name, final);
    clearNumberDraft(name);
    return;
  }
  const n = propertyFloatFromInput(raw);
  if (n != null) onItemWrite(fieldValue, name, n);
  clearNumberDraft(name);
}

function onDateWrite(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  value: unknown,
) {
  onItemWrite(fieldValue, name, propertyDateFromInput(value));
}

function onDatetimeWrite(
  fieldValue: () => WritableComputedRef<any> | { value: any },
  name: string,
  value: unknown,
) {
  onItemWrite(fieldValue, name, propertyDatetimeFromPickerModel(value));
}

async function reloadResolved() {
  const generation = ++resolveGeneration;
  const store = (binding.store ?? props.store) as WebModelStore<T> | undefined;
  const name = fieldName.value;
  if (!store || !name || typeof (store as any).ResolveProperties !== 'function') {
    resolvedItems.value = [];
    return;
  }
  const record = binding.recordRef?.()?.value ?? {};
  const map = toView(binding.fieldRef?.()?.value);
  const payload = { ...(record as any), [name]: map };
  const opts = props.containerId !== undefined ? { containerId: props.containerId } : undefined;
  try {
    const items = await (store as any).ResolveProperties(payload, name, opts);
    if (generation !== resolveGeneration) return;
    const list = Array.isArray(items) ? (items as ResolvedPropertyItem[]) : [];
    const { skipped } = filterRenderablePropertyItems(list);
    for (const item of skipped) {
      console.warn(`[ChoyPropertiesField] skipping unsupported property type '${item.type}' (${item.name})`);
    }
    resolvedItems.value = list;
  } catch (e) {
    if (generation !== resolveGeneration) return;
    console.warn('[ChoyPropertiesField] ResolveProperties failed', e);
    resolvedItems.value = [];
  }
}

watch(
  () => {
    const rec = binding.recordRef?.()?.value;
    const map = binding.fieldRef?.()?.value;
    const name = fieldName.value;
    return [
      (rec as { Id?: unknown } | null | undefined)?.Id ?? null,
      props.containerId,
      name,
      props.store ?? binding.store,
      isTableLike.value,
      JSON.stringify(map ?? null),
      propertiesRecordWatchFingerprint(rec, name),
    ] as const;
  },
  () => {
    void reloadResolved();
  },
  { immediate: true },
);
</script>
