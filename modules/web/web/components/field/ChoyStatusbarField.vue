<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FieldBase
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
    <!-- Same chrome in edit + display so clickable works outside edit mode. -->
    <template #edit="{ fieldValue, record }">
      <div
        class="choy-statusbar-field flex flex-wrap gap-1"
        role="group"
        data-testid="choy-statusbar"
        :aria-disabled="!isInteractive || pending || undefined"
      >
        <ChoyButton
          v-for="(opt, index) in optionsFor(record)"
          :key="`${opt.value}-${index}`"
          type="button"
          size="sm"
          class="choy-statusbar-opt"
          :data-value="opt.value"
          :variant="normalizeSegmentedModelValue(fieldValue().value) === opt.value ? 'default' : 'outline'"
          :disabled="!isInteractive || pending || opt.disabled"
          :aria-pressed="normalizeSegmentedModelValue(fieldValue().value) === opt.value"
          @click="onSelect(fieldValue, opt.value)"
        >
          {{ opt.label }}
        </ChoyButton>
      </div>
    </template>

    <template #display="{ fieldValue, record }">
      <div
        class="choy-statusbar-field flex flex-wrap gap-1"
        role="group"
        data-testid="choy-statusbar"
        :aria-disabled="!isInteractive || pending || undefined"
      >
        <ChoyButton
          v-for="(opt, index) in optionsFor(record)"
          :key="`${opt.value}-${index}`"
          type="button"
          size="sm"
          class="choy-statusbar-opt"
          :data-value="opt.value"
          :variant="normalizeSegmentedModelValue(fieldValue().value) === opt.value ? 'default' : 'outline'"
          :disabled="!isInteractive || pending || opt.disabled"
          :aria-pressed="normalizeSegmentedModelValue(fieldValue().value) === opt.value"
          @click="onSelect(fieldValue, opt.value)"
        >
          {{ opt.label }}
        </ChoyButton>
      </div>
    </template>
  </FieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V extends string = FieldPathType<T, P>">
import { computed, inject, onMounted, ref, type Ref } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { WritableComputedRef } from 'vue';
import FieldBase, { type FieldStateExpr, type FormItemProps } from './FieldBase.vue';
import ChoyButton from '../layout/ChoyButton.vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';
import { FIELD_PRESENTATION_FIELDS_GET_ATTRS } from '@/web/web/stores/fieldsGet';
import {
  applyStatusbarSelect,
  currentFromFieldValue,
  currentFromRowRef,
  fromStatusbarView,
  normalizeSegmentedModelValue,
  pickRootOnchangeSelection,
  resolveStatusbarOptions,
  resolveStatusbarWhitelist,
  toStatusbarView,
  validateStatusbarValue,
  type StatusbarBeforeChange,
  type StatusbarMetaOption,
  type StatusbarOption,
} from './statusbarHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/StatusbarField' });

defineOptions({ name: 'ChoyStatusbarField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    /** When true, allow writing the selection value. Default false. */
    clickable?: boolean;
    disabled?: boolean;
    /** Write gate before assigning fieldValue. */
    beforeChange?: StatusbarBeforeChange;
    /** Visible value whitelist / order. */
    statusbarVisible?: string[];
    /** Optional secondary whitelist (same semantics as SelectionField.selection). */
    selection?: string[];

    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, any>;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    agg?: NarrowAggProp<NonNumericAggFns>;
    /** Prefer `inline` in form header chrome (outside form layout). */
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
  },
);

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: any): string | null => toStatusbarView(raw);
const fromView = (v: string | null) => fromStatusbarView(v) as unknown as V;

const lastOnchangeResult = inject<Ref<any | null>>('lastOnchangeResult', ref(null));
const pending = ref(false);

const baseField = computed(() => String(binding.prop));
const leafKey = computed(() => {
  const segs = baseField.value.split('.').filter(Boolean);
  return segs.length > 0 ? segs[segs.length - 1]! : '';
});

const modelStore = computed(() => (binding.store ?? props.store) as WebModelStore<T> | undefined);

const metaOptions = computed<StatusbarMetaOption[]>(() => {
  const leaf = leafKey.value;
  const store = modelStore.value;
  const fromStore = leaf && store?.getFieldMeta ? store.getFieldMeta(leaf) : undefined;
  const meta = fromStore || binding.meta;
  const sel = meta?.selection;
  if (!Array.isArray(sel) || sel.length === 0) return [];
  return sel.map((item: { value: unknown; label?: unknown }) => {
    const value = String(item.value);
    const label = item.label == null ? value : String(item.label);
    return { value, label };
  });
});

const metaReadonly = computed(() => {
  const leaf = leafKey.value;
  const store = modelStore.value;
  const fromStore = leaf && store?.getFieldMeta ? store.getFieldMeta(leaf) : undefined;
  const meta = fromStore || binding.meta;
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
      return true;
    }
  }
  return false;
});

const isInteractive = computed(
  () => props.clickable && !props.disabled && !exprReadonly.value && !metaReadonly.value,
);

const whitelist = computed(() => resolveStatusbarWhitelist(props.statusbarVisible, props.selection));

onMounted(() => {
  const store = modelStore.value;
  const leaf = leafKey.value;
  if (!store?.ensureFieldsGet || !leaf) return;
  void store.ensureFieldsGet([leaf], [...FIELD_PRESENTATION_FIELDS_GET_ATTRS]);
});

function optionsFor(rowRef?: any): StatusbarOption[] {
  const filt = pickRootOnchangeSelection(lastOnchangeResult.value, baseField.value);
  let current = currentFromRowRef(rowRef, leafKey.value);
  if (current == null) {
    current = readFieldCurrent();
  }

  return resolveStatusbarOptions({
    meta: metaOptions.value,
    whitelist: whitelist.value,
    current,
    onchangeValues: filt?.values,
    onchangeDisabled: filt?.disabled,
  });
}

function readFieldCurrent(): string | null {
  try {
    return currentFromFieldValue(binding.fieldRef().value);
  } catch {
    return null;
  }
}

async function onSelect(getter: () => WritableComputedRef<string | null>, raw: string | number | boolean) {
  if (!isInteractive.value || pending.value) return;
  const current = getter().value != null ? String(getter().value) : null;
  pending.value = true;
  try {
    await applyStatusbarSelect({
      interactive: true,
      pending: false,
      nextRaw: raw,
      current,
      options: optionsFor(),
      beforeChange: props.beforeChange,
      write: (next) => {
        getter().value = next as any;
      },
    });
  } finally {
    pending.value = false;
  }
}

const mergedFormItemProps = computed(() => {
  const extra = props.formItemProps ?? {};
  const extraClass = (extra as { class?: unknown }).class;
  const classes = ['o-statusbar-form-item'];
  if (extraClass != null && extraClass !== '') {
    if (Array.isArray(extraClass)) classes.push(...extraClass.map(String));
    else classes.push(String(extraClass));
  }
  return {
    ...extra,
    class: classes,
  };
});

const internalRule = {
  type: 'string',
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    const err = validateStatusbarValue(value, optionsFor(), {
      mustBeString: _t('Value must be a string'),
      invalid: (v) => _t('Invalid option value: %s', v),
    });
    if (err) cb(err);
    else cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules ?? []), internalRule]);
</script>

<style lang="scss" scoped>
.o-statusbar-field {
  :deep(.o-statusbar-form-item),
  :deep(.el-form-item) {
    margin-bottom: 0;
  }

  :deep(.o-field-base__label) {
    display: none;
  }
}
</style>
