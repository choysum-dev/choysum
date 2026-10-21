<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    v-bind="$attrs"
    class="choy-statusbar-field mb-0"
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :formItemProps="mergedFormItemProps"
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
  >
    <!-- Same chrome in edit + display; interactivity follows isInteractive. -->
    <template #edit="{ fieldValue, record, inputId, describedBy }">
      <div
        class="choy-statusbar-field__control"
        role="radiogroup"
        data-testid="choy-statusbar"
        :data-clickable="isInteractive ? 'true' : undefined"
        :id="inputId || undefined"
        :aria-describedby="describedBy || undefined"
        :aria-disabled="!isInteractive || pending || undefined"
      >
        <button
          v-for="item in ribbonItems(record, fieldValue().value)"
          :key="item.opt.value"
          type="button"
          role="radio"
          class="choy-statusbar-opt"
          :data-value="item.opt.value"
          :data-selected="item.selected ? 'true' : undefined"
          :data-stage="item.stage"
          :data-edge="item.edge"
          :aria-checked="item.selected ? 'true' : 'false'"
          :disabled="pending || item.opt.disabled ? true : undefined"
          @click="onSelect(fieldValue, item.opt.value)"
        >
          {{ item.opt.label }}
        </button>
      </div>
    </template>

    <template #display="{ fieldValue, record }">
      <div
        class="choy-statusbar-field__control"
        role="radiogroup"
        data-testid="choy-statusbar"
        :data-clickable="isInteractive ? 'true' : undefined"
        :aria-disabled="!isInteractive || pending || undefined"
      >
        <button
          v-for="item in ribbonItems(record, fieldValue().value)"
          :key="item.opt.value"
          type="button"
          role="radio"
          class="choy-statusbar-opt"
          :data-value="item.opt.value"
          :data-selected="item.selected ? 'true' : undefined"
          :data-stage="item.stage"
          :data-edge="item.edge"
          :aria-checked="item.selected ? 'true' : 'false'"
          :disabled="pending || item.opt.disabled ? true : undefined"
          @click="onSelect(fieldValue, item.opt.value)"
        >
          {{ item.opt.label }}
        </button>
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V extends string = FieldPathType<T, P>">
import { computed, inject, onMounted, ref, type Ref } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { WritableComputedRef } from 'vue';
import { cn } from '@/web/web/lib/utils';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
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

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyStatusbarField' });

defineOptions({ name: 'ChoyStatusbarField', inheritAttrs: false });

type IsAny<T> = 0 extends 1 & T ? true : false;

type StatusbarStage = 'past' | 'current' | 'future';
type StatusbarEdge = 'first' | 'middle' | 'last';

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    /**
     * Write gate for stage clicks.
     * null/omit: clickable in form edit mode.
     * true: always when not readonly; false: never.
     * (null default avoids Vue Boolean-prop coercing omit → false.)
     */
    clickable?: boolean | null;
    disabled?: boolean;
    /** Write gate before assigning fieldValue. */
    beforeChange?: StatusbarBeforeChange;
    /** Visible value whitelist / order. */
    statusbarVisible?: string[];
    /** Optional secondary whitelist (same semantics as SelectionField.selection). */
    selection?: string[];

    formItemProps?: Partial<FormItemProps>;
    orientation?: 'horizontal' | 'vertical';
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
    clickable: null,
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

const isInteractive = computed(() => {
  if (props.disabled || exprReadonly.value || metaReadonly.value) return false;
  if (props.clickable === false) return false;
  if (props.clickable === true) return true;
  return binding.env?.isEditMode === true;
});

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

function isSelected(raw: unknown, value: string): boolean {
  return normalizeSegmentedModelValue(raw) === value;
}

/** Index of the current value in options; -1 when missing (all stages render as future). */
function currentIndexOf(options: StatusbarOption[], raw: unknown): number {
  const v = normalizeSegmentedModelValue(raw);
  if (!v) return -1;
  return options.findIndex((o) => o.value === v);
}

function stageFor(index: number, currentIndex: number): StatusbarStage {
  if (currentIndex < 0) return 'future';
  if (index < currentIndex) return 'past';
  if (index === currentIndex) return 'current';
  return 'future';
}

function edgeFor(index: number, len: number): StatusbarEdge {
  if (len <= 1 || index === 0) return 'first';
  if (index === len - 1) return 'last';
  return 'middle';
}

function ribbonItems(rowRef: unknown, raw: unknown) {
  const options = optionsFor(rowRef);
  const currentIndex = currentIndexOf(options, raw);
  const len = options.length;
  return options.map((opt, index) => ({
    opt,
    selected: isSelected(raw, opt.value),
    stage: stageFor(index, currentIndex),
    edge: edgeFor(index, len),
  }));
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

const mergedFormItemProps = computed(() => ({
  class: cn('choy-statusbar-form-item', props.formItemProps?.class),
}));

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
