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
      <ChoyDatetimeCell :field-value="fieldValue" :options="bufferOptions" :display-format="displayFormat" :picker-props="datePickerProps" v-bind="$attrs" :id="inputId"
        :aria-describedby="describedBy || undefined"
        :aria-invalid="invalid ? 'true' : undefined" />
    </template>
    <template #display="{ fieldValue }">
      <span class="choy-field-display-text inline-block max-w-full truncate whitespace-nowrap text-foreground">{{ toDisplayText(fieldValue().value) }}</span>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, string | Date>, V = FieldPathType<T, P>">
import { computed, defineComponent, h, type PropType } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Narrow aggregation types to count_distinct only.
import type { NarrowAggProp, TemporalAggFns } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import DateTimePicker from '../internal/DateTimePicker.vue';
import dayjs from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import { createTranslate } from '@/web/web/i18n';
import {
  formatUtcIso,
  getUserTimeZone,
  parseUtc,
  userWallDateToUtc,
  utcToUserWallDate
} from '@/web/web/utils/datetime';
dayjs.extend(customParseFormat);

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyDatetimeField' });

defineOptions({ name: 'ChoyDatetimeField' });

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
    datePickerProps?: Record<string, unknown>;
    formItemProps?: Partial<FormItemProps>;
    orientation?: 'horizontal' | 'vertical';
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
    // Temporal fields currently support count_distinct only.
    agg?: NarrowAggProp<TemporalAggFns>;
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    displayFormat: 'YYYY-MM-DD HH:mm:ss',
    valueFormat: 'YYYY-MM-DD[T]HH:mm:ss.SSSZ',
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    datePickerProps: () => ({}),
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    bufferStrategy: 'live',
    bufferIdleDelay: 150,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const displayFormat = computed(() => props.displayFormat || 'YYYY-MM-DD HH:mm:ss');
const storageFormat = computed(() => props.valueFormat || 'YYYY-MM-DD[T]HH:mm:ss.SSSZ');

function padFractionToSSS(input: string): string {
  return input.replace(/(T\d{2}:\d{2}:\d{2}\.)(\d{1,2})(?=(Z|[+-]\d{2}:\d{2})$)/, (_m, p1, frac) => p1 + (frac + '000').slice(0, 3));
}

function parseFlexible(s: string): dayjs.Dayjs | null {
  let m = parseUtc(s, storageFormat.value, true);
  if (m.isValid()) return m;

  const padded = padFractionToSSS(s);
  if (padded !== s) {
    m = parseUtc(padded, storageFormat.value, true);
    if (m.isValid()) return m;
  }

  const candidates = ['YYYY-MM-DD[T]HH:mm:ssZ', 'YYYY-MM-DD[T]HH:mm:ss.SSSZ', 'YYYY-MM-DD[T]HH:mm:ss[Z]', 'YYYY-MM-DD[T]HH:mm:ss.SSS[Z]'];
  for (const f of candidates) {
    m = parseUtc(s, f, true);
    if (m.isValid()) return m;
    if (padded !== s) {
      m = parseUtc(padded, f, true);
      if (m.isValid()) return m;
    }
  }

  m = parseUtc(s);
  return m.isValid() ? m : null;
}

const toView = (raw: any): FieldType => {
  if (raw == null) return null;
  if (raw instanceof Date) {
    return utcToUserWallDate(raw, getUserTimeZone());
  }
  if (typeof raw === 'string') {
    const m = parseFlexible(raw);
    return m ? utcToUserWallDate(m.toDate(), getUserTimeZone()) : null;
  }
  const d = new Date(raw);
  return isNaN(d.getTime()) ? null : utcToUserWallDate(d, getUserTimeZone());
};

const fromView = (v: FieldType) => {
  if (v == null) return null as any;
  const utc = userWallDateToUtc(v instanceof Date ? v : new Date(v), getUserTimeZone());
  if (!utc) return null as any;
  return (formatUtcIso(utc, storageFormat.value) ?? null) as unknown as V;
};

const toDisplayText = (v: FieldType) => {
  if (!v) return '';
  // fieldValue is a user-TZ wall-carrier Date from toView; format local components as-is.
  const m = dayjs(v);
  return m.isValid() ? m.format(displayFormat.value) : '';
};

function normalizeToDate(v: any): FieldType {
  if (v instanceof Date) return isNaN(v.getTime()) ? null : v;
  // DateTimePicker emits YYYY-MM-DDTHH:mm:ss (legacy YYYY-MM-DDTHH:mm also accepted).
  if (typeof v === 'string') {
    let parsed = dayjs(v, 'YYYY-MM-DDTHH:mm:ss', true);
    if (!parsed.isValid()) parsed = dayjs(v, 'YYYY-MM-DDTHH:mm', true);
    if (!parsed.isValid()) return null;
    return parsed.toDate();
  }
  return v ? new Date(v) : null;
}

function isValidValue(value: any): boolean {
  if (value == null || value === '') return true;
  if (value instanceof Date) return dayjs(value).isValid();
  if (typeof value === 'string') return !!parseFlexible(value);
  return dayjs(new Date(value)).isValid();
}

const internalRule = {
  validator: (_r: unknown, value: unknown, cb: (error?: Error) => void) => {
    if (!isValidValue(value)) return cb(new Error(_t('Invalid date time')));
    cb();
  },
} as RuleItem;
const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), internalRule]);

function sameDate(a: Date | null, b: Date | null) {
  if (a === b) return true;
  if (!a || !b) return false;
  return a.getTime() === b.getTime();
}

const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: FieldType) => (v && !isNaN(v.getTime()) ? v : null),
  equals: (a: FieldType, b: FieldType) => sameDate(a, b),
}));

const ChoyDatetimeCell = defineComponent({
  name: 'ChoyDatetimeCell',
  inheritAttrs: false,
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
        modelRef.value = v;
      },
      p.options
    );
    return () => {
      const current = buffer.editingValue.value;
      const value =
        current instanceof Date && !isNaN(current.getTime())
          ? dayjs(current).format('YYYY-MM-DDTHH:mm:ss')
          : null;
      return h(DateTimePicker, {
        ...attrs,
        ...(p.pickerProps || {}),
        class: 'choy-date-picker w-full',
        placeholder: _t('Select date and time'),
        modelValue: value,
        'onUpdate:modelValue': (raw: string | null) => {
          buffer.setEditing(raw ? normalizeToDate(raw) : null);
          buffer.onBlur();
        },
      });
    };
  },
});
</script>

