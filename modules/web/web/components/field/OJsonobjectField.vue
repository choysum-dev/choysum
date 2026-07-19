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
      <OJsonCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :placeholder="effectivePlaceholder"
        :autosize="autosize"
        :nullable="nullable"
        :allow-array="allowArray"
        v-bind="$attrs"
      />
    </template>

    <template #display="{ fieldValue }">
      <pre class="o-json-display">{{ displayString(fieldValue().value) }}</pre>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, Record<string, any> | any[] | null | undefined>, V = FieldPathType<T, P>">
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RuleItem } from 'async-validator';
import { ElInput, type FormItemProps } from 'element-plus';
import { ref, watch, computed, defineComponent, h } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OJsonobjectField' });

defineOptions({ name: 'OJsonobjectField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type JsonVal = Record<string, any> | any[] | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    nullable?: boolean;
    allowArray?: boolean;
    placeholder?: string;
    autosize?: boolean | { minRows?: number; maxRows?: number };

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
    // Added render mode and inline error support.
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    label: '',
    rules: () => [],
    nullable: true,
    allowArray: false,
    placeholder: '',
    autosize: () => ({ minRows: 3, maxRows: 14 }),
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    bufferStrategy: 'blur',
    bufferIdleDelay: 500,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const effectivePlaceholder = computed(() => props.placeholder || _t('Enter a JSON object'));

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

const toView = (raw: any): JsonVal => {
  if (raw == null) return null;
  if (typeof raw === 'object') return raw as any;
  try {
    const v = JSON.parse(String(raw));
    return typeof v === 'object' ? v : null;
  } catch {
    return null;
  }
};
const fromView = (v: JsonVal) => v as unknown as V;

function stableStringify(v: any): string {
  try {
    if (v == null) return '';
    if (Array.isArray(v)) {
      return JSON.stringify(v, null, 2);
    }
    return JSON.stringify(v, Object.keys(v || {}).sort(), 2);
  } catch {
    return '';
  }
}

function normalizeIncoming(v: any): JsonVal {
  if (v == null) return null;
  if (typeof v === 'object') return v as any;
  if (typeof v === 'string') {
    try {
      const parsed = JSON.parse(v);
      if (parsed === null) return null;
      if (typeof parsed === 'object') return parsed as any;
      return null;
    } catch {
      return null;
    }
  }
  return null;
}

function displayString(v: any): string {
  if (v == null) return '';
  const norm = normalizeIncoming(v);
  return norm == null ? '' : stableStringify(norm);
}

function tryParse(raw: string): { ok: boolean; val?: any; err?: string } {
  try {
    const v = JSON.parse(raw);
    if (v === null) return { ok: true, val: null };
    const isArr = Array.isArray(v);
    const isObj = typeof v === 'object' && !isArr;
    if (isArr && !props.allowArray) return { ok: false, err: _t('Arrays are not allowed') };
    if (!isArr && !isObj) return { ok: false, err: _t('Must be an object') };
    return { ok: true, val: v };
  } catch {
    return { ok: false, err: _t('JSON parse failed') };
  }
}

function jsonEquals(a: any, b: any) {
  if (a === b) return true;
  try {
    return JSON.stringify(a) === JSON.stringify(b);
  } catch {
    return false;
  }
}

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (typeof value === 'string') {
      try {
        value = JSON.parse(value);
      } catch {
        /* ignore */
      }
    }
    if (value == null) {
      if (props.nullable) return cb();
      return cb(new Error(_t('Cannot be empty')));
    }
    if (typeof value !== 'object') return cb(new Error(_t('Must be an object')));
    if (Array.isArray(value) && !props.allowArray) return cb(new Error(_t('Arrays are not allowed')));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);

const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: JsonVal) => v,
  equals: jsonEquals,
}));

const OJsonCell = defineComponent({
  name: 'OJsonCell',
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
    placeholder: String,
    autosize: [Boolean, Object],
    nullable: Boolean,
    allowArray: Boolean,
  },
  setup(p) {
    const modelRef = computed<JsonVal>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v as any;
      },
    });

    const editingText = ref<string>('');
    const textDirty = ref(false);
    const parseError = ref<string | null>(null);

    watch(
      () => modelRef.value,
      v => {
        if (textDirty.value) return;
        const norm = normalizeIncoming(v);
        if (v !== norm) modelRef.value = norm;
        editingText.value = norm == null ? '' : stableStringify(norm);
      },
      { immediate: true }
    );

    const buffer = useBufferedCommit<JsonVal>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any
    );

    function onInput(v: string) {
      editingText.value = v;
      textDirty.value = true;
      parseError.value = null;
    }
    function onBlur() {
      const raw = editingText.value;
      if (raw.trim() === '') {
        if (p.nullable) {
          buffer.setEditing(null);
          buffer.onBlur();
          textDirty.value = false;
          parseError.value = null;
          return;
        } else {
          parseError.value = _t('Cannot be empty');
          buffer.onBlur();
          return;
        }
      }
      const parsed = tryParse(raw);
      if (!parsed.ok) {
        parseError.value = parsed.err || _t('Parse failed');
        buffer.onBlur();
        return;
      }
      parseError.value = null;
      const canonical = stableStringify(parsed.val);
      editingText.value = canonical;
      textDirty.value = false;
      buffer.setEditing(parsed.val);
      buffer.onBlur();
    }

    return () =>
      h('div', {}, [
        h(ElInput, {
          class: 'o-json-input',
          type: 'textarea',
          placeholder: p.placeholder,
          autosize: p.autosize,
          modelValue: editingText.value,
          'onUpdate:modelValue': (val: any) => onInput(val),
          onBlur,
        }),
        parseError.value ? h('div', { class: 'o-json-err' }, parseError.value) : null,
      ]);
  },
});
</script>

<style scoped>
.o-json-input {
  width: 100%;
  font-family: monospace;
}
.o-json-display {
  margin: 0;
  padding: 4px 8px;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: monospace;
  line-height: 1.4;
}
.o-json-err {
  margin-top: 4px;
  font-size: 12px;
  color: var(--el-color-error);
}
</style>
