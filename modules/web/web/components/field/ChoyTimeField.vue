<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FieldBase
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
      <ChoyTimeCell :field-value="fieldValue" :options="bufferOptions" :display-format="displayFormat" :picker-props="timePickerProps" v-bind="$attrs" />
    </template>
    <template #display="{ fieldValue }">
      <span class="choy-field-display-text">{{ toDisplayText(fieldValue().value) }}</span>
    </template>
  </FieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | Date>, V = FieldPathType<T, P>">
import { computed, defineComponent, h, type PropType } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Temporal fields only support the narrowed count_distinct aggregate.
import type { NarrowAggProp, TemporalAggFns } from '@/web/web/composables/useField';
import FieldBase, { type FieldStateExpr, type FormItemProps } from './FieldBase.vue';
import dayjs from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import { createTranslate } from '@/web/web/i18n';
dayjs.extend(customParseFormat);

const { _t } = createTranslate('web', { scope: 'web/components/field/TimeField' });

defineOptions({ name: 'ChoyTimeField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type FieldType = Date | null;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    displayFormat?: string;
    valueFormat?: string;
    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;
    timePickerProps?: Record<string, unknown>;
    formItemProps?: Partial<FormItemProps>;
    vColumnProps?: Record<string, unknown>;

    bufferStrategy?: CommitStrategy;
    bufferIdleDelay?: number;
    commitOnBlur?: boolean;
    // Time fields currently only allow count_distinct.
    agg?: NarrowAggProp<TemporalAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    displayFormat: 'HH:mm:ss',
    valueFormat: 'HH:mm:ss',
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    timePickerProps: () => ({}),
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    bufferStrategy: 'live',
    bufferIdleDelay: 120,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const displayFormat = computed(() => props.displayFormat || 'HH:mm:ss');
const storageFormat = computed(() => props.valueFormat || 'HH:mm:ss');

// Parse strings into Dayjs, preferring the storage format before fallback candidates.
function parseFlexible(s: string): dayjs.Dayjs | null {
  let m = dayjs(s, storageFormat.value, true);
  if (m.isValid()) return m;

  const candidates = ['HH:mm:ss', 'HH:mm'];
  for (const f of candidates) {
    m = dayjs(s, f, true);
    if (m.isValid()) return m;
  }
  m = dayjs(s);
  return m.isValid() ? m : null;
}

// string/Date/number -> Date|null
const toView = (raw: any): FieldType => {
  if (raw == null) return null;
  if (raw instanceof Date) return isNaN(raw.getTime()) ? null : raw;
  if (typeof raw === 'string') {
    const m = parseFlexible(raw);
    return m ? m.toDate() : null;
  }
  const d = new Date(raw);
  return isNaN(d.getTime()) ? null : d;
};

// Convert Date|null back into the storage-format string.
const fromView = (v: FieldType) => {
  if (v == null) return null as any;
  const m = v instanceof Date ? dayjs(v) : dayjs(new Date(v));
  if (!m.isValid()) return null as any;
  return m.format(storageFormat.value) as unknown as V;
};

const toDisplayText = (v: FieldType) => {
  if (!v) return '';
  const m = dayjs(v);
  return m.isValid() ? m.format(displayFormat.value) : '';
};

function normalizeToDate(v: any): FieldType {
  return v instanceof Date ? (isNaN(v.getTime()) ? null : v) : v ? new Date(v) : null;
}

function isValidValue(value: any): boolean {
  if (value == null || value === '') return true;
  if (value instanceof Date) return dayjs(value).isValid();
  if (typeof value === 'string') return !!parseFlexible(value);
  return dayjs(new Date(value)).isValid();
}

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (!isValidValue(value)) return cb(new Error(_t('Invalid time value')));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);

function sameTime(a: Date | null, b: Date | null) {
  if (a === b) return true;
  if (!a || !b) return false;
  return a.getTime() === b.getTime();
}

// Match ODateTime and ODate by buffering Date|null at the view layer while storing strings underneath.
const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: FieldType) => (v && !isNaN(v.getTime()) ? v : null),
  equals: (a: FieldType, b: FieldType) => sameTime(a, b),
}));

const ChoyTimeCell = defineComponent({
  name: 'ChoyTimeCell',
  props: {
    fieldValue: { type: Function as PropType<() => { value: any }>, required: true },
    options: { type: Object as PropType<any>, required: true },
    displayFormat: { type: String, required: true },
    pickerProps: { type: Object as PropType<Record<string, any>>, default: () => ({}) },
  },
  setup(p, { attrs }) {
    const modelRef = computed<FieldType>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v;
      },
    });

    const buffer = useBufferedCommit<FieldType>(
      () => modelRef.value,
      v => {
        (p.fieldValue as any)().value = v;
      },
      p.options
    );

    return () => {
      const current = buffer.editingValue.value;
      const value =
        current instanceof Date && !isNaN(current.getTime())
          ? dayjs(current).format('HH:mm:ss')
          : '';
      return h('input', {
        ...attrs,
        ...(p.pickerProps || {}),
        type: 'time',
        step: 1,
        class: 'choy-time-picker',
        value,
        onInput: (e: Event) => {
          const raw = (e.target as HTMLInputElement).value;
          buffer.setEditing(raw ? normalizeToDate(raw) : null);
        },
        onBlur: () => buffer.onBlur(),
      });
    };
  },
});
</script>

<style scoped>
.choy-field-display-text {
  line-height: var(--el-component-size-base, 32px);
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding: 0 11px;
}
.choy-time-picker {
  width: 100%;
}
</style>
