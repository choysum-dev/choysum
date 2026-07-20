<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
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

    <template #display="{ fieldValue, inputName, inputId }">
      <div class="o-bool-editor">
        <el-switch
          v-if="widget === 'switch'"
          class="o-bool-input"
          :name="inputName"
          :id="inputId"
          v-bind="switchProps"
          :model-value="fieldValue().value === true"
          :disabled="true"
          :active-value="true"
          :inactive-value="false"
          :active-text="switchActiveText"
          :inactive-text="switchInactiveText"
        />
        <el-checkbox
          v-else
          class="o-bool-input"
          :name="inputName"
          :id="inputId"
          v-bind="checkboxProps"
          :model-value="fieldValue().value === true"
          :disabled="true"
          :indeterminate="fieldValue().value === null && !(nullAsFalse || !nullable)"
        >
          {{ checkboxLabel }}
        </el-checkbox>
      </div>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, boolean>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { ElSwitch, ElCheckbox, ElButton, type FormItemProps } from 'element-plus';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
// Narrow aggregation types to count_distinct only.
import type { NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { useBufferedCommit, type CommitStrategy } from '@/web/web/composables/useBufferedCommit';
import { createTranslate } from '@/web/web/i18n';
import { computed, defineComponent, h } from 'vue';

const { _t } = createTranslate('web', { scope: 'web/components/field/OBooleanField' });

defineOptions({ name: 'OBooleanField' });

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
    switchProps?: Partial<InstanceType<typeof ElSwitch>['$props']>;
    checkboxProps?: Partial<InstanceType<typeof ElCheckbox>['$props']>;

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
    return () =>
      h('div', { class: 'o-bool-editor' }, [
        p.widget === 'checkbox'
          ? h(
              ElCheckbox,
              {
                ...(p.checkboxProps as any),
                ...attrs,
                class: 'o-bool-input',
                name: p.inputName,
                id: p.inputId,
                modelValue: buffer.editingValue.value === true,
                indeterminate: buffer.editingValue.value === null && !(p.nullAsFalse || !p.nullable),
                'onUpdate:modelValue': (val: any) => setVal(val),
              },
              () => p.checkboxLabel
            )
          : h(ElSwitch, {
              ...(p.switchProps as any),
              ...attrs,
              class: 'o-bool-input',
              name: p.inputName,
              id: p.inputId,
              modelValue: buffer.editingValue.value === true,
              activeValue: true,
              inactiveValue: false,
              activeText: p.switchActiveText,
              inactiveText: p.switchInactiveText,
              'onUpdate:modelValue': (val: any) => setVal(val),
            }),
        clearable
          ? h(
              ElButton,
              {
                link: true,
                type: 'primary',
                class: 'o-clear-btn',
                onClick: () => {
                  if (buffer.editingValue.value !== null) {
                    buffer.setEditing(null);
                    buffer.onBlur(); // Commit immediately instead of waiting for focus loss.
                  }
                },
              },
              () => _t('Clear')
            )
          : null,
      ]);
  },
});
</script>

<style scoped>
.o-bool-editor {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0 11px;
}
.o-bool-input {
  vertical-align: middle;
}
.o-clear-btn {
  padding: 0;
}
</style>
