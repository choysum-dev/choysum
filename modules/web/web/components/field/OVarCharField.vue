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
    <!-- Reuse the slot-provided fieldValue for both form and row rendering. -->
    <template #edit="{ fieldValue }">
      <OVarCharCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="placeholder"
        :maxlength="effectiveMaxLength ?? undefined"
        :show-word-limit="showWordLimit"
        v-bind="$attrs"
      />
    </template>
    <template #display="{ fieldValue }">
      <span class="o-field-display-text">{{ toDisplayText(fieldValue().value) }}</span>
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
// Non-numeric varchar aggregates are narrowed to count_distinct.
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { useBufferedCommit, CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OVarCharField' });

defineOptions({ name: 'OVarCharField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type ViewType = string | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    maxLength?: number;
    nullable?: boolean;
    trimOnBlur?: 'none' | 'both';
    placeholder?: string;
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
    // Only count_distinct is supported for varchar aggregates.
    agg?: NarrowAggProp<NonNumericAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    nullable: true,
    trimOnBlur: 'none',
    placeholder: '',
    showWordLimit: true,

    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,

    formItemProps: () => ({}),
    vColumnProps: () => ({}),

    bufferStrategy: 'idle',
    bufferIdleDelay: 360,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;
const metaSize = computed<number | undefined>(() => binding.meta?.size as any);
const effectiveMaxLength = computed(() => props.maxLength ?? metaSize.value ?? undefined);

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

/**
 * Delay buffered-commit creation until the cell component has access to fieldValue.
 */
const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: string | null) => {
    if (v == null) return null;
    let s = String(v);
    s = cutToLen(s, effectiveMaxLength.value);
    if (props.trimOnBlur === 'both') s = s.trim();
    if (s === '' && props.nullable) return null;
    return s;
  },
  equals: (a: string | null, b: string | null) => a === b,
}));

/**
 * Internal cell component with per-cell buffering backed by OFieldBase fieldValue.
 */
const OVarCharCell = defineComponent({
  name: 'OVarCharCell',
  props: {
    fieldValue: { type: Function, required: true }, // () => WritableComputedRef<ViewType>
    options: { type: Object, required: true },
    placeholder: String,
    maxlength: [Number, String],
    showWordLimit: Boolean,
  },
  setup(p, { attrs }) {
    // Adapt useBufferedCommit to the slot-provided getter and setter.
    const modelRef = computed<ViewType>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v as any;
      },
    });

    const buffer = useBufferedCommit<ViewType>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any
    );

    function onBlur() {
      // trimOnBlur is already handled inside normalize.
      buffer.onBlur();
    }

    return () =>
      h(ElInput, {
        ...attrs,
        class: 'o-input',
        placeholder: p.placeholder,
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
      return cb(new Error(_t('Value is required')));
    }
    if (typeof value !== 'string') return cb(new Error(_t('Value must be a string')));
    const n = strLen(value);
    const N = effectiveMaxLength.value;
    if (N != null && n > N) return cb(new Error(_t('Length must not exceed %s', N)));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

<style lang="scss" scoped>
.o-field-display-text {
  line-height: var(--el-component-size-base, 32px);
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 11px;
}
.o-input {
  width: 100%;
}
</style>
