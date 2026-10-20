<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
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
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue, inputId, describedBy, invalid}">
      <ChoyBigintCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="placeholder"
        :wire-format="wireFormat"
        :nullable="nullable"
        :min="min"
        :max="max"
        v-bind="$attrs"
        :id="inputId"
        :aria-describedby="describedBy || undefined"
        :aria-invalid="invalid ? 'true' : undefined"
      />
    </template>
    <template #display="{ fieldValue }">
      <div
        v-if="isEmptyDisplay(fieldValue().value) && showFormEmptyPlaceholder"
        class="choy-bigint-display choy-bigint-display--empty text-sm text-muted-foreground text-end"
      >
        {{ emptyDisplayText }}
      </div>
      <span
        v-else
        class="choy-field-display-text choy-field-display-text--numeric inline-block max-w-full truncate whitespace-nowrap text-foreground text-end"
        >{{ toDisplayText(fieldValue().value) }}</span
      >
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | number | bigint | null>, V = FieldPathType<T, P>">
import { ref, computed, watch, defineComponent, h, type PropType } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField, AggProp } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import Input from '../vendor/ui/input/Input.vue';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import {
  formatGroupedNumericString,
  isFormEmptyPlaceholderMode,
  resolveNumericInputEl,
  resyncControlledNumericRaw,
} from './numericFieldHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyBigintField' });

defineOptions({ name: 'ChoyBigintField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

/* ---------------- Types ---------------- */
type ViewType = string | null;

/* ---------------- Constants ---------------- */
const INT64_MAX = BigInt('9223372036854775807');
const INT64_MIN = BigInt('-9223372036854775808');
const JS_MAX_SAFE = BigInt(Number.MAX_SAFE_INTEGER);
const JS_MIN_SAFE = -JS_MAX_SAFE;

/* ---------------- Props ---------------- */
const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    wireFormat?: 'string' | 'number';
    min?: string | number | bigint;
    max?: string | number | bigint;
    nullable?: boolean;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    placeholder?: string;

    formItemProps?: Partial<FormItemProps>;
    orientation?: 'horizontal' | 'vertical';
    vColumnProps?: {
      width?: number | string;
      minWidth?: number;
      align?: 'left' | 'center' | 'right';
      fixed?: 'left' | 'right';
      sortable?: boolean;
    };

    bufferStrategy?: CommitStrategy; // live | idle | blur
    bufferIdleDelay?: number;
    commitOnBlur?: boolean;
    agg?: AggProp;
    // Added render mode and inline error support.
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    wireFormat: 'string',
    min: undefined,
    max: undefined,
    nullable: true,
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    placeholder: '',
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    bufferStrategy: 'idle',
    bufferIdleDelay: 300,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

/* ---------------- Binding ---------------- */
const binding = (props.binding ??
  useField<T, P, V>({
    store: props.store as WebModelStore<T>,
    prop: props.prop as P,
    agg: props.agg,
  })) as UseField<T, V>;

const showFormEmptyPlaceholder = computed(() =>
  isFormEmptyPlaceholderMode(props.renderMode, binding.env?.isForm)
);
const emptyDisplayText = computed(() => _t('No content'));

/* ---------------- Helpers ---------------- */
function toBigInt(v: string | number | bigint | undefined): bigint | null {
  if (v === undefined || v === null) return null;
  try {
    if (typeof v === 'bigint') return v;
    if (typeof v === 'number') return Number.isInteger(v) ? BigInt(v) : null;
    return BigInt(String(v).trim());
  } catch {
    return null;
  }
}
const minBI = computed<bigint>(() => toBigInt(props.min) ?? INT64_MIN);
const maxBI = computed<bigint>(() => toBigInt(props.max) ?? INT64_MAX);

function parseStrict(s: string): bigint | null {
  const t = s.trim();
  if (!/^-?\d+$/.test(t)) return null;
  try {
    return BigInt(t);
  } catch {
    return null;
  }
}

function clampValue(n: bigint): bigint {
  if (n < minBI.value) return minBI.value;
  if (n > maxBI.value) return maxBI.value;
  return n;
}

function isIntermediateInput(s: string | null): boolean {
  return s === '' || s === '-';
}

/* View-layer conversion for the cell buffer. */
const toView = (raw: any): ViewType => {
  if (raw == null || raw === '') return null;
  if (typeof raw === 'string') {
    return /^\s*-?\d+\s*$/.test(raw) ? raw.trim() : null;
  }
  if (typeof raw === 'number') return Number.isInteger(raw) ? String(raw) : null;
  if (typeof raw === 'bigint') return raw.toString();
  return null;
};

const fromView = (v: ViewType) => {
  if (v == null || v === '') return null as any;
  const bi = parseStrict(v);
  if (bi == null) return null as any;
  if (props.wireFormat === 'number') {
    if (bi > JS_MAX_SAFE || bi < JS_MIN_SAFE) return null as any;
    return Number(bi) as unknown as V;
  }
  return bi.toString() as unknown as V;
};

function toDisplayText(v: any) {
  if (v == null || v === '') return '';
  const text = typeof v === 'string' ? v : String(v);
  try {
    return formatGroupedNumericString(text, useI18nStore().currentLocale?.numberFormat);
  } catch {
    return text;
  }
}
function isEmptyDisplay(v: any): boolean {
  return v == null || v === '';
}

/* ---------------- Buffered Commit Options ---------------- */
const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: ViewType) => {
    if (v == null || v === '') return null;
    if (isIntermediateInput(v)) return null;
    const bi = parseStrict(v);
    if (bi == null) return null;
    let clamped = clampValue(bi);
    if (props.wireFormat === 'number') {
      if (clamped > JS_MAX_SAFE) clamped = JS_MAX_SAFE;
      if (clamped < JS_MIN_SAFE) clamped = JS_MIN_SAFE;
    }
    return clamped.toString();
  },
  equals: (a: ViewType, b: ViewType) => a === b,
}));

/* ---------------- Cell Component ---------------- */
const ChoyBigintCell = defineComponent({
  name: 'ChoyBigintCell',
  props: {
    fieldValue: { type: Function as PropType<() => { value: any }>, required: true },
    options: { type: Object as PropType<any>, required: true },
    placeholder: { type: String, default: '' },
    wireFormat: { type: String as PropType<'string' | 'number'>, default: 'string' },
    nullable: { type: Boolean, default: true },
    // Accept string or number input, then normalize internally to BigInt.
    min: { type: [String, Number] as unknown as PropType<string | number | bigint>, default: undefined },
    max: { type: [String, Number] as unknown as PropType<string | number | bigint>, default: undefined },
  },
  setup(props, { attrs }) {
    const modelRef = computed<any>({
      get: () => (props.fieldValue as any)().value,
      set: v => {
        (props.fieldValue as any)().value = v;
      },
    });

    const editingRaw = ref<ViewType>(null);
    const inputHost = ref<unknown>(null);

    watch(
      () => modelRef.value,
      nv => {
        if (!isIntermediateInput(editingRaw.value)) editingRaw.value = toView(nv);
      },
      { immediate: true }
    );

    const buffer = useBufferedCommit<ViewType>(
      () => modelRef.value, // Use the view value directly.
      v => {
        modelRef.value = v; // Assign back directly; ChoyFieldBase writes through fromView.
      },
      props.options
    );

    function onInput(raw: string) {
      if (!/^[-]?\d*$/.test(raw)) {
        resyncControlledNumericRaw(editingRaw, resolveNumericInputEl(inputHost.value));
        return;
      }
      editingRaw.value = raw;
      if (raw === '' || raw === '-') return;
      const bi = parseStrict(raw);
      if (bi == null) return;
      buffer.setEditing(clampValue(bi).toString());
    }

    function onBlur() {
      const raw = editingRaw.value;
      if (raw == null || raw === '' || raw === '-') {
        buffer.setEditing(null);
        buffer.onBlur();
        return;
      }
      const bi = parseStrict(raw);
      if (bi != null) buffer.setEditing(clampValue(bi).toString());
      buffer.onBlur();
    }

    return () =>
      h(Input, {
        ...attrs,
        ref: inputHost,
        class: 'choy-bigint-input w-full text-end',
        type: 'text',
        modelValue: editingRaw.value ?? '',
        placeholder: props.placeholder,
        inputmode: 'numeric',
        'onUpdate:modelValue': (v: string) => onInput(v ?? ''),
        onBlur,
      });
  },
});

/* ---------------- Validation ---------------- */
function isValidValue(value: any): string | null {
  if (value == null || value === '') {
    if (props.nullable) return null;
    return _t('Cannot be empty');
  }
  let bi: bigint | null = null;
  if (typeof value === 'string') bi = parseStrict(value);
  else if (typeof value === 'number') {
    if (!Number.isInteger(value)) return _t('Must be an integer');
    try {
      bi = BigInt(value);
    } catch {
      bi = null;
    }
  } else if (typeof value === 'bigint') bi = value;
  if (bi == null) return _t('Must be a valid integer');
  if (bi < minBI.value) return _t('Must not be less than %s', minBI.value.toString());
  if (bi > maxBI.value) return _t('Must not be greater than %s', maxBI.value.toString());
  if (props.wireFormat === 'number' && (bi > JS_MAX_SAFE || bi < JS_MIN_SAFE)) return _t('Exceeds safe integer range');
  return null;
}

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    const msg = isValidValue(value);
    if (msg) return cb(new Error(msg));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

