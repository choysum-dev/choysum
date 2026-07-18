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
      <OIntCell :field-value="fieldValue" :options="bufferOptions" :placeholder="placeholder" :nullable="nullable" :min="min" :max="max" v-bind="$attrs" />
    </template>
    <template #display="{ fieldValue }">
      <span class="o-field-display-text">{{ fieldValue().value == null ? '' : fieldValue().value }}</span>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, number | null>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElInput, type FormItemProps } from 'element-plus';
import { ref, watch, computed, defineComponent, h } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { AggProp } from '@/web/web/composables/useField';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OIntField' });

defineOptions({ name: 'OIntField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type FieldType = number | null;
type ViewType = FieldType;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    min?: number;
    max?: number;
    nullable?: boolean;
    placeholder?: string;

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, any>;

    bufferStrategy?: CommitStrategy;
    bufferIdleDelay?: number;
    commitOnBlur?: boolean;
    // Aggregate support.
    agg?: AggProp;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    label: '',
    rules: () => [],
    min: Number.MIN_SAFE_INTEGER,
    max: Number.MAX_SAFE_INTEGER,
    nullable: true,
    placeholder: '',
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    bufferStrategy: 'idle',
    bufferIdleDelay: 260,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

// Binding with agg forwarded to useField.
const binding = (props.binding ??
  useField<T, P, V>({
    store: props.store as WebModelStore<T>,
    prop: props.prop as P,
    agg: props.agg,
  })) as UseField<T, V>;

const toView = (raw: any): ViewType => {
  if (raw == null || raw === '') return null;
  const n = Number(raw);
  return Number.isFinite(n) ? n : null;
};
const fromView = (v: ViewType) => v as unknown as V;

function parseStrict(s: string): number | null {
  if (!/^[-]?\d+$/.test(s)) return null;
  const n = Number(s);
  return Number.isSafeInteger(n) ? n : null;
}
function clampValue(n: number): number {
  if (n < props.min!) n = props.min!;
  if (n > props.max!) n = props.max!;
  return n;
}
function isIntermediateInput(s: string | null): boolean {
  return s === '' || s === '-';
}

const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: FieldType) => {
    if (v == null) return null;
    return clampValue(v);
  },
  equals: (a: FieldType, b: FieldType) => a === b,
}));

const OIntCell = defineComponent({
  name: 'OIntCell',
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
    placeholder: String,
    nullable: Boolean,
    min: Number,
    max: Number,
  },
  setup(p, { attrs }) {
    const modelRef = computed<FieldType>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v;
      },
    });

    const editingRaw = ref<string | null>(null);
    watch(
      () => modelRef.value,
      nv => {
        if (isIntermediateInput(editingRaw.value)) return;
        editingRaw.value = nv == null ? null : String(nv);
      },
      { immediate: true }
    );

    const buffer = useBufferedCommit<FieldType>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any
    );

    function onInput(raw: string) {
      if (!/^[-]?\d*$/.test(raw)) return;
      editingRaw.value = raw;
      if (raw === '' || raw === '-') return;
      const n = parseStrict(raw);
      if (n == null) return;
      buffer.setEditing(clampValue(n));
    }
    function onBlur() {
      const raw = editingRaw.value;
      if (raw == null || raw === '' || raw === '-') {
        if (p.nullable) buffer.setEditing(null);
        buffer.onBlur();
        return;
      }
      const n = parseStrict(raw);
      if (n != null) buffer.setEditing(clampValue(n));
      buffer.onBlur();
    }

    return () =>
      h(ElInput, {
        ...attrs,
        class: 'o-input o-int-input',
        modelValue: editingRaw.value,
        placeholder: p.placeholder,
        inputmode: 'numeric',
        'onUpdate:modelValue': (val: any) => onInput(val),
        onBlur,
      });
  },
});

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (value == null) {
      if (props.nullable) return cb();
      return cb(new Error(_t('Cannot be empty')));
    }
    if (typeof value !== 'number' || !Number.isSafeInteger(value)) {
      return cb(new Error(_t('Must be an integer')));
    }
    if (value < props.min! || value > props.max!) {
      return cb(new Error(_t('Range %s ~ %s', props.min, props.max)));
    }
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

<style scoped lang="scss">
.o-field-display-text {
  line-height: var(--el-component-size-base, 32px);
  padding: 0 11px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.o-input {
  width: 100%;
}
</style>
