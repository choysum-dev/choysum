<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
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
      <OBigintCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="placeholder"
        :wire-format="wireFormat"
        :nullable="nullable"
        :min="min"
        :max="max"
        v-bind="$attrs"
      />
    </template>
    <template #display="{ fieldValue }">
      <span class="o-field-display-text">{{ toDisplayText(fieldValue().value) }}</span>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | number | bigint | null>, V = FieldPathType<T, P>">
import { ref, computed, watch, defineComponent, h, type PropType } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElInput, type FormItemProps } from 'element-plus';
import { useField } from '@/web/web/composables/useField';
import type { UseField, AggProp } from '@/web/web/composables/useField';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';

defineOptions({ name: 'OBigintField' });

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
    label: '',
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
  return typeof v === 'string' ? v : String(v);
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
const OBigintCell = defineComponent({
  name: 'OBigintCell',
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
        modelRef.value = v; // Assign back directly; OFieldBase writes through fromView.
      },
      props.options
    );

    function onInput(raw: string) {
      if (!/^[-]?\d*$/.test(raw)) return;
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
      h(ElInput, {
        ...attrs,
        class: 'o-input o-bigint-input',
        modelValue: editingRaw.value,
        placeholder: props.placeholder,
        inputmode: 'numeric',
        'onUpdate:modelValue': (val: any) => onInput(val),
        onBlur,
      });
  },
});

/* ---------------- Validation ---------------- */
function isValidValue(value: any): string | null {
  if (value == null || value === '') {
    if (props.nullable) return null;
    return '不能为空';
  }
  let bi: bigint | null = null;
  if (typeof value === 'string') bi = parseStrict(value);
  else if (typeof value === 'number') {
    if (!Number.isInteger(value)) return '必须是整数';
    try {
      bi = BigInt(value);
    } catch {
      bi = null;
    }
  } else if (typeof value === 'bigint') bi = value;
  if (bi == null) return '必须是有效整数';
  if (bi < minBI.value) return `不能小于 ${minBI.value.toString()}`;
  if (bi > maxBI.value) return `不能大于 ${maxBI.value.toString()}`;
  if (props.wireFormat === 'number' && (bi > JS_MAX_SAFE || bi < JS_MIN_SAFE)) return '超出数值安全范围';
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

<style scoped lang="scss">
.o-bigint-input {
  width: 100%;
}
.o-field-display-text {
  line-height: var(--el-component-size-base, 32px);
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 11px;
}
</style>
