<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="choy-search-filter__row flex items-center gap-2">
    <select
      class="w-field w-[180px]"
      :placeholder="_t('Field')"
      :value="condition.field"
      @change="onFieldChange(($event.target as HTMLSelectElement).value)"
    >
      <option v-for="f in fields" :key="f.prop" :label="f.label" :value="f.prop" />
    </select>

    <select
      class="w-operator w-[140px]"
      :disabled="!condition.field"
      :value="condition.operator"
      @change="onOperatorChange(($event.target as HTMLSelectElement).value)"
    >
      <option v-for="op in operatorOptions" :key="op.value" :label="op.label" :value="op.value" />
    </select>

    <input
      v-if="condition.field && isMultiValueOperator(condition.operator) && !isRelationValueField"
      class="w-value flex-1"
      data-multi="true"
      :value="(multiValues || []).join(',')"
      :placeholder="_t('Add values')"
      @change="onMultiValuesChange(String(($event.target as HTMLInputElement).value).split(',').map(s => s.trim()).filter(Boolean))"
    />
    <component
      v-else-if="condition.field && requiresValue(condition.operator)"
      :key="`${conditionId}-${condition.field}-${condition.operator}`"
      :is="fieldComponent"
      class="flex-1"
      :store="store"
      :binding="binding"
      v-bind="extraProps"
      :label="''"
      :rules="[]"
      :placeholder="valuePlaceholder"
      :formItemProps="{ labelWidth: 0, style: { margin: 0, padding: 0 } }"
    />
    <span v-else-if="condition.field && isNullOperator(condition.operator)" class="w-value choy-null-flag flex-1 text-info">NULL</span>
    <input v-else class="flex-1" :placeholder="_t('Select a field')" disabled />

    <ChoyButton class="rm" size="sm" variant="destructive" @click="onRemove">{{ _t('Remove') }}</ChoyButton>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import type { Condition } from '@/web/web/query/types';
import type { WebFieldMetadata, WebModelStore } from '@/web/web/stores/modelStore';
import { useStandaloneField } from '@/web/web/composables/useField';
import { useInjectedFilterEditorBindings } from '@/web/web/composables/search/useFilterEditorBindings';
import { createTranslate } from '@/web/web/i18n';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';

import ChoyVarcharField from '@/web/web/components/field/ChoyVarcharField.vue';
import ChoyTextField from '@/web/web/components/field/ChoyTextField.vue';
import ChoyIntField from '@/web/web/components/field/ChoyIntField.vue';
import ChoyBigintField from '@/web/web/components/field/ChoyBigintField.vue';
import ChoyNumberField from '@/web/web/components/field/ChoyNumberField.vue';
import ChoyDecimalField from '@/web/web/components/field/ChoyDecimalField.vue';
import ChoyMonetaryField from '@/web/web/components/field/ChoyMonetaryField.vue';
import ChoyBooleanField from '@/web/web/components/field/ChoyBooleanField.vue';
import ChoyDateField from '@/web/web/components/field/ChoyDateField.vue';
import ChoyTimeField from '@/web/web/components/field/ChoyTimeField.vue';
import ChoyDatetimeField from '@/web/web/components/field/ChoyDatetimeField.vue';
import ChoyJsonField from '@/web/web/components/field/ChoyJsonField.vue';
import ChoyManyToOneField from '@/web/web/components/field/ChoyManyToOneField.vue';
import ChoyManyToOneRefField from '@/web/web/components/field/ChoyManyToOneRefField.vue';
import ChoyBinaryField from '@/web/web/components/field/ChoyBinaryField.vue';
import ChoyImageField from '@/web/web/components/field/ChoyImageField.vue';
import ChoySelectionField from '@/web/web/components/field/ChoySelectionField.vue';

defineOptions({ name: 'SearchFilterCondition' });

const { _t } = createTranslate('web', { scope: 'web/components/view/search/SearchFilterCondition' });

type CondLike = Condition & { tempId?: string };

const props = defineProps<{
  condition: CondLike;
  fields: Array<{ prop: string; label: string }>;
  store: WebModelStore<any>;
  onUpdateCondition: (id: string, patch: Partial<CondLike>) => void;
  onRemoveCondition: (id: string) => void;
}>();

const { metaTypeOf, relationStoreOf, getOperatorOptionsForField, isNullOperator, requiresValue, defaultValueFor, isMultiValueOperator } =
  useInjectedFilterEditorBindings(props.store);

const conditionId = computed(() => props.condition.tempId || props.condition.id);

const operatorOptions = computed(() => getOperatorOptionsForField(props.condition.field));

const fieldType = computed(() => metaTypeOf(props.condition.field || ''));
const isRelationValueField = computed(() => fieldType.value === 'manytoone' || fieldType.value === 'manytooneref');

const fieldComponent = computed(() => {
  switch (fieldType.value) {
    case 'char':
    case 'varchar':
    case 'html':
      // Search filter uses plaintext entry for html; Form uses ChoyHtmlField.
      return ChoyVarcharField;
    case 'text':
      return ChoyTextField;
    case 'int':
      return ChoyIntField;
    case 'bigint':
      return ChoyBigintField;
    case 'number':
      return ChoyNumberField;
    case 'decimal':
      return ChoyDecimalField;
    case 'monetary':
      return ChoyMonetaryField;
    case 'boolean':
      return ChoyBooleanField;
    case 'date':
      return ChoyDateField;
    case 'time':
      return ChoyTimeField;
    case 'datetime':
      return ChoyDatetimeField;
    case 'jsonobject':
      return ChoyJsonField;
    case 'manytoone':
      return ChoyManyToOneField;
    case 'manytooneref':
      return ChoyManyToOneRefField;
    case 'binary':
      return ChoyBinaryField;
    case 'image':
      return ChoyImageField;
    case 'selection':
      return ChoySelectionField;
    default:
      return ChoyVarcharField;
  }
});

const valuePlaceholder = computed(() => {
  switch (fieldType.value) {
    case 'manytoone':
    case 'manytooneref':
      return _t('Select a record');
    case 'date':
      return _t('Select date');
    case 'datetime':
      return _t('Select date and time');
    case 'time':
      return _t('Select time');
    case 'jsonobject':
      return _t('Enter JSON');
    case 'selection':
      return _t('Please select...');
    default:
      return _t('Value');
  }
});

const extraProps = computed(() => {
  if (fieldType.value !== 'manytoone' && fieldType.value !== 'manytooneref') {
    return {};
  }
  return {
    toView: (raw: any) => {
      if (raw == null) return null;
      if (typeof raw === 'object') return raw;
      return { Id: raw };
    },
    fromView: (v: any) => {
      if (v == null) return null;
      return typeof v === 'object' ? (v.Id ?? null) : v;
    },
  };
});

const multiValues = computed(() => {
  const v = props.condition.value;
  if (Array.isArray(v)) return v.map(x => String(x ?? '')).filter(Boolean);
  if (v == null || v === '') return [] as string[];
  return [String(v)];
});

function onMultiValuesChange(next: string[]) {
  props.onUpdateCondition(conditionId.value, { value: Array.isArray(next) ? next : [] });
}

const valueRef = computed({
  get: () => props.condition.value,
  set: v => props.onUpdateCondition(conditionId.value, { value: v }),
});

const fieldMeta = computed(() => {
  const fieldName = props.condition.field || '';
  const t = metaTypeOf(fieldName);
  const staticMeta = ((props.store as any)?.fieldsMetadata?.[fieldName] || {}) as Partial<WebFieldMetadata>;
  return {
    ...staticMeta,
    type: (staticMeta.type || t) as any,
    // Filter value editors must stay writable even when the column is form-readonly
    // (e.g. DisplayName / computed fields) — otherwise FieldBase swaps to an empty display slot.
    isReadonly: false,
  } as Partial<WebFieldMetadata>;
});

// Built once per condition row instance (not per parent re-render).
const binding = useStandaloneField({
  value: valueRef,
  meta: fieldMeta.value,
  prop: props.condition.field || 'value',
  env: { isForm: true, isEditMode: true, viewMode: 'edit' },
  record: {},
}) as any;

// Keep ensureFieldsGet for selection/ACL overlays, but never let getFieldMeta re-apply isReadonly.
binding.store = {
  get fieldsMetadata() {
    return (props.store as any)?.fieldsMetadata;
  },
  getFieldMeta(name: string) {
    const raw =
      typeof (props.store as any)?.getFieldMeta === 'function'
        ? (props.store as any).getFieldMeta(name)
        : (props.store as any)?.fieldsMetadata?.[name];
    return raw ? { ...raw, isReadonly: false } : raw;
  },
  ensureFieldsGet: (...args: any[]) => (props.store as any)?.ensureFieldsGet?.(...args),
  getFieldsGetTranslatedString: (name: string) => (props.store as any)?.getFieldsGetTranslatedString?.(name),
};

watch(
  () => [props.condition.field, fieldMeta.value] as const,
  ([fieldName, meta]) => {
    binding.meta = meta;
    binding.prop = fieldName || 'value';
    binding.relationStore =
      metaTypeOf(fieldName) === 'manytoone' || metaTypeOf(fieldName) === 'manytooneref'
        ? relationStoreOf(fieldName)
        : undefined;
  },
  { immediate: true }
);

function onFieldChange(val: string) {
  const ops = getOperatorOptionsForField(val);
  const firstOp = ops.length > 0 ? ops[0].value : undefined;
  props.onUpdateCondition(conditionId.value, { field: val, operator: firstOp, value: undefined });
}

function onOperatorChange(op: string) {
  const patch: Partial<CondLike> = { operator: op };
  if (isNullOperator(op)) {
    patch.value = null;
  } else if (isMultiValueOperator(op)) {
    const cur = props.condition.value;
    if (!Array.isArray(cur)) {
      patch.value = cur == null || cur === '' ? [] : [cur];
    }
  } else if (requiresValue(op) && (props.condition.value === undefined || props.condition.value === null)) {
    const dv = defaultValueFor(metaTypeOf(props.condition.field || ''));
    if (dv !== undefined) patch.value = dv;
  }
  props.onUpdateCondition(conditionId.value, patch);
}

function onRemove() {
  props.onRemoveCondition(conditionId.value);
}
</script>

