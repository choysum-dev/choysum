<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :toView="toView"
    :fromView="fromView"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue }">
      <ONumberCell :field-value="fieldValue" :options="bufferOptions" :placeholder="placeholder" :nullable="nullable" :min="min" :max="max" v-bind="$attrs" />
    </template>
    <template #display="{ fieldValue }">
      <span class="o-field-display-text">{{ toDisplayText(fieldValue().value) }}</span>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, number | null>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ref, watch, computed, defineComponent, h } from 'vue';
import { ElInput } from 'element-plus';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { AggProp } from '@/web/web/composables/useField';
import OFieldBase from './OFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';

defineOptions({ name: 'ONumberField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type FieldType = number | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    nullable?: boolean;
    min?: number;
    max?: number;
    placeholder?: string;
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
    nullable: true,
    bufferStrategy: 'idle',
    bufferIdleDelay: 300,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

// Binding with agg forwarded to useField.
const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: any): FieldType => (raw === '' || raw == null ? null : typeof raw === 'number' && Number.isFinite(raw) ? raw : null);
const fromView = (v: FieldType) => v as V;

function toDisplayText(v: any) {
  return v == null ? '' : String(v);
}

function isIntermediateInput(s: string | null): boolean {
  if (s == null) return false;
  if (s === '-' || s === '.' || s === '-.') return true;
  if (/^-?\d+\.$/.test(s)) return true;
  return false;
}

function parseStrict(s: string): number | null {
  if (s === '' || s == null) return null;
  const n = Number(s);
  return Number.isFinite(n) ? n : null;
}

const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: number | null) => {
    if (v == null) return null;
    let n = v;
    if (props.min !== undefined && n < props.min) n = props.min;
    if (props.max !== undefined && n > props.max) n = props.max;
    return n;
  },
  equals: (a: number | null, b: number | null) => a === b,
}));

const ONumberCell = defineComponent({
  name: 'ONumberCell',
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
        (p.fieldValue as any)().value = v as any;
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
      if (raw === '' && p.nullable) {
        editingRaw.value = null;
        buffer.setEditing(null);
        return;
      }
      if (!/^[-]?\d*(\.\d*)?$/.test(raw)) return;
      editingRaw.value = raw;
      if (isIntermediateInput(raw)) return;
      const n = parseStrict(raw);
      if (n == null) return;
      buffer.setEditing(n);
    }
    function onBlur() {
      const raw = editingRaw.value;
      if (raw == null) {
        if (p.nullable) buffer.setEditing(null);
        buffer.onBlur();
        return;
      }
      if (isIntermediateInput(raw)) {
        const final = parseStrict(raw.replace(/\.$/, ''));
        if (final == null) {
          if (p.nullable) buffer.setEditing(null);
        } else {
          buffer.setEditing(final);
        }
        buffer.onBlur();
        return;
      }
      const n = parseStrict(raw);
      if (n != null) buffer.setEditing(n);
      buffer.onBlur();
    }
    return () =>
      h(ElInput, {
        ...attrs,
        class: 'o-input o-number-input',
        modelValue: editingRaw.value,
        placeholder: p.placeholder,
        inputmode: 'decimal',
        'onUpdate:modelValue': (val: any) => onInput(val),
        onBlur,
      });
  },
});

const internalRule = {
  type: 'number',
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (value == null) {
      return props.nullable ? cb() : cb(new Error('Value is required'));
    }
    if (typeof value !== 'number' || !Number.isFinite(value)) return cb(new Error('Value must be a number'));
    if (props.min !== undefined && value < props.min) return cb(new Error(`Value must be at least ${props.min}`));
    if (props.max !== undefined && value > props.max) return cb(new Error(`Value must be at most ${props.max}`));
    cb();
  },
  trigger: 'blur',
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

<style scoped lang="scss">
.o-field-display-text {
  line-height: 32px;
  padding: 0 11px;
  text-align: right;
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
}
.o-number-input {
  :deep(.el-input__inner) {
    text-align: right;
  }
  :deep(.el-input__wrapper input) {
    text-align: right;
  } /* Compatible with the newer Element Plus input structure. */
}
</style>
