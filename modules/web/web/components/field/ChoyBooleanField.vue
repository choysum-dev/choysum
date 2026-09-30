<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="rules"
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
  >
    <template #edit="{ fieldValue, inputName, inputId }">
      <OBooleanCell
        :field-value="fieldValue"
        :options="bufferOptions"
        :widget="widget"
        :nullable="nullable"
        :clearable="clearable"
        :null-as-false="nullAsFalse"
        :switch-active-text="switchActiveText"
        :switch-inactive-text="switchInactiveText"
        :checkbox-label="checkboxLabel"
        :switch-props="switchProps"
        :checkbox-props="checkboxProps"
        :input-name="inputName"
        :input-id="inputId"
        v-bind="$attrs"
      />
    </template>

    <template #display="{ fieldValue }">
      <span class="choy-bool-display text-sm text-foreground" data-testid="choy-bool-display">
        {{ displayLabel(fieldValue().value) }}
      </span>
    </template>
  </FieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, boolean>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Narrow aggregation types to count_distinct only.
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import FieldBase, { type FieldStateExpr, type FormItemProps } from './FieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import { createTranslate } from '@/web/web/i18n';
import { computed, defineComponent, h } from 'vue';

const { _t } = createTranslate('web', { scope: 'web/components/field/BooleanField' });

defineOptions({ name: 'ChoyBooleanField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

type FieldType = boolean | null;
type WidgetType = 'switch' | 'checkbox';

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];

    widget?: WidgetType;
    nullable?: boolean;
    clearable?: boolean;
    nullAsFalse?: boolean;

    switchActiveText?: string;
    switchInactiveText?: string;
    checkboxLabel?: string;

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
    switchProps?: Record<string, unknown>;
    checkboxProps?: Record<string, unknown>;

    bufferStrategy?: CommitStrategy;
    bufferIdleDelay?: number;
    commitOnBlur?: boolean;
    // Only count_distinct is supported.
    agg?: NarrowAggProp<NonNumericAggFns>;
    // Added render mode and inline error support.
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    widget: 'switch',
    nullable: false,
    clearable: false,
    nullAsFalse: false,
    switchActiveText: '',
    switchInactiveText: '',
    checkboxLabel: '',
    formItemProps: () => ({}),
    vColumnProps: () => ({}),
    switchProps: () => ({}),
    checkboxProps: () => ({}),
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    bufferStrategy: 'idle',
    bufferIdleDelay: 160,
    commitOnBlur: true,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ??
  useField<T, P, V>({
    store: props.store as WebModelStore<T>,
    prop: props.prop as P,
    // Forward agg to useField.
    agg: props.agg,
  })) as UseField<T, V>;

const toView = (raw: any): FieldType => {
  if (raw === true || raw === false) return raw;
  if (raw == null) return null;
  if (typeof raw === 'number') return raw === 1 ? true : raw === 0 ? false : null;
  if (typeof raw === 'string') {
    const s = raw.trim().toLowerCase();
    if (['true', '1', 'yes', 'y'].includes(s)) return true;
    if (['false', '0', 'no', 'n'].includes(s)) return false;
    return null;
  }
  return null;
};
const fromView = (v: FieldType): V => {
  if (props.nullAsFalse) return !!v as unknown as V;
  return (v == null ? null : !!v) as unknown as V;
};

/** Display mode: plain text — never a disabled input wall. */
function displayLabel(value: FieldType): string {
  if (value === true) return props.switchActiveText || props.checkboxLabel || _t('Yes');
  if (value === false) return props.switchInactiveText || _t('No');
  return _t('—');
}

function toBool(v: boolean | string | number): boolean {
  return v === true || v === 'true' || v === 1;
}

const bufferOptions = computed(() => ({
  strategy: props.bufferStrategy!,
  idleDelay: props.bufferIdleDelay,
  commitOnBlur: props.commitOnBlur,
  normalize: (v: FieldType) => {
    if (v == null && props.nullable && !props.nullAsFalse) return null;
    return v === true ? true : v === false ? false : props.nullAsFalse ? false : null;
  },
  equals: (a: FieldType, b: FieldType) => a === b,
}));

const OBooleanCell = defineComponent({
  name: 'OBooleanCell',
  props: {
    fieldValue: { type: Function, required: true },
    options: { type: Object, required: true },
    widget: String,
    nullable: Boolean,
    clearable: Boolean,
    nullAsFalse: Boolean,
    switchActiveText: String,
    switchInactiveText: String,
    checkboxLabel: String,
    switchProps: Object,
    checkboxProps: Object,
    inputName: String,
    inputId: String,
  },
  setup(p, { attrs }) {
    const modelRef = computed<FieldType>({
      get: () => (p.fieldValue as any)().value,
      set: v => {
        (p.fieldValue as any)().value = v as any;
      },
    });
    const buffer = useBufferedCommit<FieldType>(
      () => modelRef.value,
      v => {
        modelRef.value = v;
      },
      p.options as any
    );
    const setVal = (v: any) => buffer.setEditing(v === true);
    const clearable = p.nullable && p.clearable && !p.nullAsFalse;
    return () => {
      const checked = buffer.editingValue.value === true;
      const indeterminate = buffer.editingValue.value === null && !(p.nullAsFalse || !p.nullable);
      return h('div', { class: 'choy-bool-editor inline-flex items-center gap-2 px-[11px]' }, [
        h('input', {
          ...(p.widget === 'checkbox' ? (p.checkboxProps as any) : (p.switchProps as any)),
          ...attrs,
          type: 'checkbox',
          class: 'choy-bool-input align-middle',
          name: p.inputName,
          id: p.inputId,
          checked,
          indeterminate,
          onChange: (e: Event) => setVal((e.target as HTMLInputElement).checked),
        }),
        clearable
          ? h(
              'button',
              {
                type: 'button',
                class: 'choy-clear-btn p-0',
                onClick: () => {
                  if (buffer.editingValue.value !== null) {
                    buffer.setEditing(null);
                    buffer.onBlur();
                  }
                },
              },
              p.checkboxLabel || _t('Clear')
            )
          : null,
      ]);
    };
  },
});
</script>

