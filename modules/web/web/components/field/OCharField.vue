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
      <OCharCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="placeholder"
        :maxlength="effectiveLength ?? undefined"
        :show-word-limit="showWordLimit"
        v-bind="$attrs"
      />
    </template>
    <template #display="{ fieldValue }">
      <span class="o-field-display-text">{{ toDisplayText(fieldValue().value) }}</span>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElInput, type FormItemProps } from 'element-plus';
import { computed, defineComponent, h } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';

defineOptions({ name: 'OCharField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type ViewType = string | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    // Field length, preferring meta.size when available.
    length?: number; // Target length for CHAR(N).
    exactLength?: boolean; // Whether the value must be exactly N characters.
    nullable?: boolean; // Whether an empty string is normalized to null.
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
    bufferStrategy?: CommitStrategy; // Buffered commit strategy: live, idle, or blur.
    bufferIdleDelay?: number; // Delay used by the idle commit strategy.
    commitOnBlur?: boolean; // Whether blur triggers a commit.
    // Only count_distinct is allowed for character aggregates.
    agg?: NarrowAggProp<NonNumericAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    label: '',
    rules: () => [],
    exactLength: false,
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
    bufferIdleDelay: 320,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const metaSize = computed<number | undefined>(() => binding.meta?.size as any);
const effectiveLength = computed(() => metaSize.value ?? props.length ?? undefined);

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

const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: string | null) => {
    if (v == null) return null;
    let s = String(v);
    s = cutToLen(s, effectiveLength.value);
    if (props.trimOnBlur === 'both') s = s.trim();
    if (s === '' && props.nullable) return null;
    return s;
  },
  equals: (a: string | null, b: string | null) => a === b,
}));

const OCharCell = defineComponent({
  name: 'OCharCell',
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
    placeholder: String,
    maxlength: [Number, String],
    showWordLimit: Boolean,
  },
  setup(p, { attrs }) {
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
    const onBlur = () => buffer.onBlur();
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
      return cb(new Error('Value is required'));
    }
    if (typeof value !== 'string') return cb(new Error('Value must be a string'));
    const n = strLen(value);
    const N = effectiveLength.value;
    if (N != null) {
      if (props.exactLength) {
        if (n !== N) return cb(new Error(`Length must equal ${N}`));
      } else {
        if (n > N) return cb(new Error(`Length must not exceed ${N}`));
      }
    }
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
