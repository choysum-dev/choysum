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
      <OTextCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="placeholder"
        :rows="rows"
        :autosize="autosize"
        :maxlength="maxLength ?? undefined"
        :show-word-limit="showWordLimit && !!maxLength"
        v-bind="$attrs"
      />
    </template>

    <template #display="{ fieldValue }">
      <div class="o-textfield-display">{{ toDisplayText(fieldValue().value) }}</div>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | null | undefined>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElInput, type FormItemProps } from 'element-plus';
import { computed, defineComponent, h } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Narrow aggregation types to count_distinct only.
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OTextField' });

defineOptions({ name: 'OTextField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type ViewType = string | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    maxLength?: number; // Optional length limit for text input.
    nullable?: boolean; // Empty strings map to null by default.
    trimOnBlur?: 'none' | 'both';
    placeholder?: string;

    rows?: number;
    autosize?: boolean | { minRows?: number; maxRows?: number };
    showWordLimit?: boolean;

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
    bufferStrategy?: CommitStrategy;
    bufferIdleDelay?: number;
    commitOnBlur?: boolean;
    // Only count_distinct is supported for text fields.
    agg?: NarrowAggProp<NonNumericAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    nullable: true,
    trimOnBlur: 'none',
    placeholder: '',
    rows: 4,
    autosize: () => ({ minRows: 3, maxRows: 10 }),
    showWordLimit: true,

    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,

    formItemProps: () => ({}),
    vColumnProps: () => ({}),

    bufferStrategy: 'blur',
    bufferIdleDelay: 400,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: any): ViewType => (raw == null ? null : String(raw));
const fromView = (v: ViewType) => v as unknown as V;
const toDisplayText = (v: ViewType) => (v == null ? '' : String(v));

function strLen(s: string) {
  return Array.from(s).length;
}
function cutToLen(s: string, max?: number) {
  if (!max || max < 0) return s;
  const arr = Array.from(s);
  return arr.slice(0, max).join('');
}

// Use the configurable buffer strategy from props.
const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: string | null) => {
    if (v == null) return null;
    let s = String(v);
    s = cutToLen(s, props.maxLength);
    if (props.trimOnBlur === 'both') s = s.trim();
    if (s === '' && props.nullable) return null;
    return s;
  },
  equals: (a: string | null, b: string | null) => a === b,
}));

const OTextCell = defineComponent({
  name: 'OTextCell',
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
    placeholder: String,
    rows: Number,
    autosize: [Boolean, Object],
    maxlength: [Number, String],
    showWordLimit: Boolean,
  },
  setup(p, { attrs }) {
    const modelRef = computed<ViewType>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v;
      },
    });
    const buffer = useBufferedCommit<ViewType>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any
    );
    const onBlur = () => buffer.onBlur();
    return () =>
      h(ElInput, {
        ...attrs,
        type: 'textarea',
        class: 'o-textarea',
        placeholder: p.placeholder,
        rows: p.rows,
        autosize: p.autosize,
        maxlength: p.maxlength,
        showWordLimit: p.showWordLimit,
        modelValue: buffer.editingValue.value,
        'onUpdate:modelValue': (val: any) => buffer.setEditing(val ?? ''),
        onBlur,
      });
  },
});

const internalRule = {
  type: 'string',
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (value == null) {
      if (props.nullable) return cb();
      return cb(new Error(_t('Cannot be empty')));
    }
    if (typeof value !== 'string') return cb(new Error(_t('Must be a string')));
    if (props.maxLength != null) {
      const n = strLen(value);
      if (n > props.maxLength) return cb(new Error(_t('Length must not exceed %s', props.maxLength)));
    }
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

<style lang="scss" scoped>
.o-textarea {
  width: 100%;
}
.o-textfield-display {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: var(--el-component-size-base, 32px);
  color: var(--el-text-color-primary);
  padding: 0 11px;
}
</style>
