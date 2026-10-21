<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
    :orientation="orientation"
    :toView="toView"
    :fromView="fromView"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue, inputId, describedBy, invalid}">
      <ChoyNumberCell :field-value="fieldValue" :options="bufferOptions" :placeholder="placeholder" :nullable="nullable" :min="min" :max="max" v-bind="$attrs" :id="inputId"
        :aria-describedby="describedBy || undefined"
        :aria-invalid="invalid ? 'true' : undefined" />
    </template>
    <template #display="{ fieldValue }">
      <div
        v-if="isEmptyDisplay(fieldValue().value) && showFormEmptyPlaceholder"
        class="choy-number-display choy-number-display--empty text-sm text-muted-foreground text-end"
      >
        {{ emptyDisplayText }}
      </div>
      <span
        v-else
        class="choy-field-display-text choy-field-display-text--numeric truncate whitespace-nowrap text-foreground inline-block max-w-full text-end"
        >{{ toDisplayText(fieldValue().value) }}</span
      >
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, number | null>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ref, watch, computed, defineComponent, h } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import type { AggProp } from '@/web/web/composables/useField';
import ChoyFieldBase from './ChoyFieldBase.vue';
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

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyNumberField' });

defineOptions({ name: 'ChoyNumberField' });

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
    orientation?: 'horizontal' | 'vertical';
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
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

const showFormEmptyPlaceholder = computed(() =>
  isFormEmptyPlaceholderMode(props.renderMode, binding.env?.isForm)
);
const emptyDisplayText = computed(() => _t('No content'));

const toView = (raw: any): FieldType => (raw === '' || raw == null ? null : typeof raw === 'number' && Number.isFinite(raw) ? raw : null);
const fromView = (v: FieldType) => v as V;

function toDisplayText(v: any) {
  if (v == null) return '';
  const text = String(v);
  try {
    return formatGroupedNumericString(text, useI18nStore().currentLocale?.numberFormat);
  } catch {
    return text;
  }
}
function isEmptyDisplay(v: any): boolean {
  return v == null || v === '';
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

const ChoyNumberCell = defineComponent({
  name: 'ChoyNumberCell',
  inheritAttrs: false,
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
    const inputHost = ref<unknown>(null);
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
      if (!/^[-]?\d*(\.\d*)?$/.test(raw)) {
        resyncControlledNumericRaw(editingRaw, resolveNumericInputEl(inputHost.value));
        return;
      }
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
      h(Input, {
        ...attrs,
        ref: inputHost,
        class: 'choy-number-input w-full text-end',
        modelValue: editingRaw.value ?? '',
        placeholder: p.placeholder,
        inputmode: 'decimal',
        'onUpdate:modelValue': (v: string) => onInput(v ?? ''),
        onBlur,
      });
  },
});

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (value == null) {
      return props.nullable ? cb() : cb(new Error(_t('Value is required')));
    }
    if (typeof value !== 'number' || !Number.isFinite(value)) return cb(new Error(_t('Value must be a number')));
    if (props.min !== undefined && value < props.min) return cb(new Error(_t('Value must be at least %s', props.min)));
    if (props.max !== undefined && value > props.max) return cb(new Error(_t('Value must be at most %s', props.max)));
    cb();
  },
  trigger: 'blur',
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);
</script>

