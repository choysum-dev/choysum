<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    v-bind="$attrs"
    class="o-statusbar-field"
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :formItemProps="mergedFormItemProps"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
  >
    <!-- Same chrome in edit + display so clickable works outside edit mode (D3). -->
    <template #edit="{ fieldValue, record }">
      <el-segmented
        class="o-statusbar"
        :model-value="normalizeValue(fieldValue().value)"
        :options="segmentedOptions(record)"
        :disabled="!isInteractive || pending"
        size="default"
        @update:model-value="(v: string | number | boolean) => onSelect(fieldValue, v)"
      />
    </template>

    <template #display="{ fieldValue, record }">
      <el-segmented
        class="o-statusbar"
        :model-value="normalizeValue(fieldValue().value)"
        :options="segmentedOptions(record)"
        :disabled="!isInteractive || pending"
        size="default"
        @update:model-value="(v: string | number | boolean) => onSelect(fieldValue, v)"
      />
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V extends string = FieldPathType<T, P>">
import { computed, inject, onMounted, ref, type Ref } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElSegmented, type FormItemProps } from 'element-plus';
import type { WritableComputedRef } from 'vue';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';
import { FIELD_PRESENTATION_FIELDS_GET_ATTRS } from '@/web/web/stores/fieldsGet';
import {
  gateBeforeChange,
  resolveStatusbarOptions,
  type StatusbarBeforeChange,
  type StatusbarMetaOption,
  type StatusbarOption,
} from './ostatusbar_helpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/OStatusbarField' });

defineOptions({ name: 'OStatusbarField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    /** When true, allow writing the selection value. Default false (D3). */
    clickable?: boolean;
    disabled?: boolean;
    /** Write gate before assigning fieldValue (D7). */
    beforeChange?: StatusbarBeforeChange;
    /** Visible value whitelist / order (D5). */
    statusbarVisible?: string[];
    /** Optional secondary whitelist (same semantics as OSelectionField.selection). */
    selection?: string[];

    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, any>;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    agg?: NarrowAggProp<NonNumericAggFns>;
    /** Prefer `inline` in form header chrome (outside el-form). */
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    clickable: false,
    disabled: false,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    renderMode: 'inline',
    showInlineError: false,
    label: '',
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: any): string | null => (raw == null ? null : String(raw));
const fromView = (v: string | null) => (v ?? null) as unknown as V;

const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));
const pending = ref(false);

const baseField = computed(() => String(binding.prop));
const leafKey = computed(() => {
  const segs = baseField.value.split('.').filter(Boolean);
  return segs[segs.length - 1] || '';
});

const modelStore = computed(() => (binding.store || props.store) as WebModelStore<T> | undefined);

const metaOptions = computed<StatusbarMetaOption[]>(() => {
  const leaf = leafKey.value;
  const store = modelStore.value;
  const meta = (leaf && store?.getFieldMeta?.(leaf)) || binding.meta;
  const sel = meta?.selection;
  if (!Array.isArray(sel) || sel.length === 0) return [];
  return sel.map((item: { value: unknown; label?: unknown }) => ({
    value: String(item.value),
    label: String(item.label ?? item.value),
  }));
});

const metaReadonly = computed(() => {
  const leaf = leafKey.value;
  const store = modelStore.value;
  const meta = (leaf && store?.getFieldMeta?.(leaf)) || binding.meta;
  return meta?.isReadonly === true;
});

const exprReadonly = computed(() => {
  const r = props.readonly;
  if (typeof r === 'boolean') return r;
  if (typeof r === 'function') {
    try {
      const record = binding.recordRef().value as T;
      const value = (binding.fieldRef().value ?? null) as V | null;
      return !!r({ record, value, env: binding.env });
    } catch {
      // Fail closed: indeterminate readonly must not become writable.
      return true;
    }
  }
  return false;
});

const isInteractive = computed(() => props.clickable && !props.disabled && !exprReadonly.value && !metaReadonly.value);

const whitelist = computed(() => {
  if (Array.isArray(props.statusbarVisible) && props.statusbarVisible.length > 0) return props.statusbarVisible;
  if (Array.isArray(props.selection) && props.selection.length > 0) return props.selection;
  return null;
});

onMounted(() => {
  const store = modelStore.value;
  const leaf = leafKey.value;
  if (!store?.ensureFieldsGet || !leaf) return;
  void store.ensureFieldsGet([leaf], [...FIELD_PRESENTATION_FIELDS_GET_ATTRS]);
});

function pickOnchangeSelection(): { values: string[]; disabled?: string[] } | null {
  const raw = lastOnchangeResult.value?.selection || [];
  if (!Array.isArray(raw) || !raw.length) return null;
  const key = baseField.value;
  const m = raw.find((s: any) => s && s.field === key);
  if (!m) return null;
  return {
    values: Array.isArray(m.selection) ? m.selection : [],
    disabled: Array.isArray(m.disabled) ? m.disabled : undefined,
  };
}

function normalizeValue(raw: unknown): string | undefined {
  if (raw == null || raw === '') return undefined;
  return String(raw);
}

function optionsFor(rowRef?: any): StatusbarOption[] {
  const filt = pickOnchangeSelection();
  let current: string | null = null;
  try {
    if (rowRef != null) {
      const row = typeof rowRef === 'function' ? rowRef() : rowRef;
      const rec = row && typeof row === 'object' && 'value' in row ? (row as any).value : row;
      const leaf = leafKey.value;
      if (rec && leaf && rec[leaf] != null) current = String(rec[leaf]);
    }
  } catch {
    current = null;
  }
  if (current == null) {
    try {
      const v = binding.fieldRef().value;
      current = v != null && v !== '' ? String(v) : null;
    } catch {
      current = null;
    }
  }

  return resolveStatusbarOptions({
    meta: metaOptions.value,
    whitelist: whitelist.value,
    current,
    onchangeValues: filt?.values,
    onchangeDisabled: filt?.disabled,
  });
}

function segmentedOptions(rowRef?: any) {
  return optionsFor(rowRef).map(o => ({
    label: o.label,
    value: o.value,
    disabled: o.disabled,
  }));
}

async function onSelect(getter: () => WritableComputedRef<string | null>, raw: string | number | boolean) {
  if (!isInteractive.value || pending.value) return;
  const next = raw == null ? '' : String(raw);
  if (!next) return;
  const current = getter().value != null ? String(getter().value) : null;
  if (next === current) return;

  const allowed = optionsFor().some(o => o.value === next && !o.disabled);
  if (!allowed) return;

  pending.value = true;
  try {
    const ok = await gateBeforeChange(props.beforeChange, next, current);
    if (ok) getter().value = next as any;
  } finally {
    pending.value = false;
  }
}

const mergedFormItemProps = computed(() => ({
  class: 'o-statusbar-form-item',
  ...(props.formItemProps || {}),
}));

const internalRule = {
  type: 'string',
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    // Treat null/undefined/'' as unset (empty is stripped from option pools).
    if (value == null || value === '') return cb();
    if (typeof value !== 'string') return cb(new Error(_t('Value must be a string')));
    const ok = optionsFor().some(o => o.value === value);
    if (!ok) return cb(new Error(_t('Invalid option value: %s', value)));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

<style lang="scss" scoped>
.o-statusbar-field {
  :deep(.o-statusbar-form-item),
  :deep(.el-form-item) {
    margin-bottom: 0;
  }

  :deep(.el-form-item__label) {
    display: none;
  }
}

.o-statusbar {
  --o-statusbar-chevron: 10px;
  --o-statusbar-pad-x: 14px;
  background: transparent !important;
  padding: 0 !important;
  border-radius: 0 !important;
  min-height: 28px;
  height: auto;

  :deep(.el-segmented__item-selected) {
    display: none !important;
  }

  :deep(.el-segmented__group) {
    display: flex;
    align-items: stretch;
    gap: 0;
    background: transparent;
  }

  :deep(.el-segmented__item) {
    position: relative;
    flex: 0 0 auto;
    margin-inline-start: calc(var(--o-statusbar-chevron) * -1);
    padding: 4px calc(var(--o-statusbar-pad-x) + var(--o-statusbar-chevron)) 4px
      calc(var(--o-statusbar-pad-x) + var(--o-statusbar-chevron));
    border-radius: 0 !important;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    font-weight: 400;
    line-height: 20px;
    clip-path: polygon(
      0 0,
      calc(100% - var(--o-statusbar-chevron)) 0,
      100% 50%,
      calc(100% - var(--o-statusbar-chevron)) 100%,
      0 100%,
      var(--o-statusbar-chevron) 50%
    );
  }

  :deep(.el-segmented__item:first-child) {
    margin-inline-start: 0;
    padding-left: var(--o-statusbar-pad-x);
    clip-path: polygon(
      0 0,
      calc(100% - var(--o-statusbar-chevron)) 0,
      100% 50%,
      calc(100% - var(--o-statusbar-chevron)) 100%,
      0 100%
    );
    border-start-start-radius: 2px !important;
    border-end-start-radius: 2px !important;
  }

  :deep(.el-segmented__item.is-selected) {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
    font-weight: 600;
    box-shadow: inset 0 0 0 1px var(--el-color-primary);
  }

  :deep(.el-segmented__item.is-disabled) {
    opacity: 1;
    cursor: default;
  }

  :deep(.el-segmented__item:not(.is-selected).is-disabled) {
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
  }

  :deep(.el-segmented__item.is-selected.is-disabled) {
    background: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
  }

  :deep(.el-segmented__item:not(.is-disabled):not(.is-selected):hover) {
    color: var(--el-color-primary);
  }

  :deep(.el-segmented__item-label) {
    overflow: visible;
  }
}
</style>
