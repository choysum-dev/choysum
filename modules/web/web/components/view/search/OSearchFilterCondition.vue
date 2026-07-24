<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-search-filter__row">
    <el-select
      class="w-field"
      :placeholder="_t('Field')"
      :model-value="condition.field"
      filterable
      clearable
      @update:model-value="onFieldChange"
    >
      <el-option v-for="f in fields" :key="f.prop" :label="f.label" :value="f.prop" />
    </el-select>

    <el-select
      class="w-operator"
      :placeholder="_t('Operator')"
      :disabled="!condition.field"
      :model-value="condition.operator"
      clearable
      @update:model-value="onOperatorChange"
    >
      <el-option v-for="op in operatorOptions" :key="op.value" :label="op.label" :value="op.value" />
    </el-select>

    <component
      v-if="condition.field && requiresValue(condition.operator)"
      :is="fieldComponent"
      class="w-value"
      :store="store"
      :binding="binding"
      v-bind="extraProps"
      :label="''"
      :rules="[]"
      :placeholder="valuePlaceholder"
      :formItemProps="{ labelWidth: 0, style: { margin: 0, padding: 0 } }"
    />
    <span v-else-if="condition.field && isNullOperator(condition.operator)" class="w-value o-null-flag">NULL</span>
    <el-input v-else class="w-value" :placeholder="_t('Select a field')" disabled />

    <el-button type="danger" text size="small" @click="onRemove">{{ _t('Remove') }}</el-button>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue';
import type { Condition } from '@/web/web/query/types';
import type { WebFieldMetadata, WebModelStore } from '@/web/web/stores/modelStore';
import { useStandaloneField } from '@/web/web/composables/useField';
import { useInjectedFilterEditorBindings } from '@/web/web/composables/search/useFilterEditorBindings';
import { createTranslate } from '@/web/web/i18n';

import OCharField from '@/web/web/components/field/OCharField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBigintField from '@/web/web/components/field/OBigintField.vue';
import ONumberField from '@/web/web/components/field/ONumberField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateField from '@/web/web/components/field/ODateField.vue';
import OTimeField from '@/web/web/components/field/OTimeField.vue';
import ODatetimeField from '@/web/web/components/field/ODatetimeField.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OBinaryField from '@/web/web/components/field/OBinaryField.vue';
import OImageField from '@/web/web/components/field/OImageField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';

defineOptions({ name: 'OSearchFilterCondition' });

const { _t } = createTranslate('web', { scope: 'web/components/view/search/OSearchFilterCondition' });

type CondLike = Condition & { tempId?: string };

const props = defineProps<{
  condition: CondLike;
  fields: Array<{ prop: string; label: string }>;
  store: WebModelStore<any>;
  onUpdateCondition: (id: string, patch: Partial<CondLike>) => void;
  onRemoveCondition: (id: string) => void;
}>();

const { metaTypeOf, relationStoreOf, getOperatorOptionsForField, isNullOperator, requiresValue, defaultValueFor } =
  useInjectedFilterEditorBindings(props.store);

const conditionId = computed(() => props.condition.tempId || props.condition.id);

const operatorOptions = computed(() => getOperatorOptionsForField(props.condition.field));

const fieldType = computed(() => metaTypeOf(props.condition.field || ''));

const fieldComponent = computed(() => {
  switch (fieldType.value) {
    case 'char':
      return OCharField;
    case 'varchar':
      return OVarCharField;
    case 'text':
      return OTextField;
    case 'int':
      return OIntField;
    case 'bigint':
      return OBigintField;
    case 'number':
      return ONumberField;
    case 'decimal':
      return ODecimalField;
    case 'boolean':
      return OBooleanField;
    case 'date':
      return ODateField;
    case 'time':
      return OTimeField;
    case 'datetime':
      return ODatetimeField;
    case 'jsonobject':
      return OJsonobjectField;
    case 'manytoone':
      return OManyToOneField;
    case 'binary':
      return OBinaryField;
    case 'image':
      return OImageField;
    case 'selection':
      return OSelectionField;
    default:
      return OVarCharField;
  }
});

const valuePlaceholder = computed(() => {
  switch (fieldType.value) {
    case 'manytoone':
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
  if (fieldType.value !== 'manytoone') return {};
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

binding.store = props.store;

watch(
  () => [props.condition.field, fieldMeta.value] as const,
  ([fieldName, meta]) => {
    binding.meta = meta;
    binding.prop = fieldName || 'value';
    binding.relationStore = metaTypeOf(fieldName) === 'manytoone' ? relationStoreOf(fieldName) : undefined;
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

<style scoped lang="scss">
.o-search-filter__row {
  display: flex;
  gap: 8px;
  align-items: center;
  .w-field {
    width: 180px;
  }
  .w-operator {
    width: 140px;
  }
  .w-value {
    flex: 1;
  }
  .o-null-flag {
    color: var(--el-color-info);
  }
}
</style>
