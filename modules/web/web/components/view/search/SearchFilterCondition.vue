<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="choy-search-filter__row grid min-w-0 items-center gap-2 rounded-md px-2 py-2 hover:bg-muted/25"
    :style="incompleteRowStyle()"
    data-slot="search-filter-row"
    :data-incomplete="incomplete ? 'true' : 'false'"
  >
    <Combobox
      class="choy-search-filter__field min-w-0"
      :model-value="condition.field || undefined"
      v-model:open="fieldOpen"
      ignore-filter
      open-on-focus
      open-on-click
      @update:model-value="onFieldCombobox"
    >
      <ComboboxAnchor class="w-full min-w-0">
        <ComboboxInput
          v-model="fieldQuery"
          class="choy-search-filter__field-input h-control w-full min-w-0 bg-background text-sm font-medium"
          data-testid="choy-search-filter-field"
          :placeholder="_t('Field')"
          :aria-label="_t('Field')"
          :display-value="fieldDisplayValue"
          @focus="onFieldFocus"
        />
      </ComboboxAnchor>
      <ComboboxContent class="choy-search-filter__field-content z-[80] p-0">
        <div class="max-h-56 overflow-y-auto p-1">
          <div
            v-if="!filteredFields.length"
            class="px-2 py-1.5 text-sm text-muted-foreground"
          >
            {{ _t('No fields') }}
          </div>
          <ComboboxItem
            v-for="f in filteredFields"
            :key="f.prop"
            :value="f.prop"
            class="text-sm"
          >
            {{ f.label }}
          </ComboboxItem>
        </div>
      </ComboboxContent>
    </Combobox>

    <Select
      class="choy-search-filter__operator min-w-0"
      :model-value="condition.operator || null"
      @update:model-value="onOperatorSelect"
    >
      <SelectTrigger
        class="choy-search-filter__operator-trigger h-control w-full min-w-0 bg-background text-muted-foreground"
        :disabled="!condition.field"
        :placeholder="_t('Operator')"
        :aria-label="_t('Operator')"
      />
      <SelectContent>
        <SelectItem v-for="op in operatorOptions" :key="op.value" :value="op.value">
          {{ op.label }}
        </SelectItem>
      </SelectContent>
    </Select>

    <div class="choy-search-filter__value w-value min-w-0">
      <Input
        v-if="condition.field && isMultiValueOperator(condition.operator) && !isRelationValueField"
        class="w-value h-control w-full bg-background"
        data-multi="true"
        :model-value="(multiValues || []).join(',')"
        :placeholder="_t('Add values')"
        @update:model-value="onMultiInput"
      />
      <component
        v-else-if="condition.field && requiresValue(condition.operator)"
        :key="`${conditionId}-${condition.field}-${condition.operator}`"
        :is="fieldComponent"
        class="choy-search-filter__value-field w-full min-w-0"
        :store="store"
        :binding="binding"
        v-bind="extraProps"
        :label="''"
        :rules="[]"
        :placeholder="valuePlaceholder"
        :formItemProps="{ labelWidth: 0, style: { margin: 0, padding: 0 } }"
      />
      <span
        v-else-if="condition.field && isNullOperator(condition.operator)"
        class="choy-null-flag flex h-control items-center rounded-md border border-dashed border-border bg-background px-2 text-sm text-muted-foreground"
      >
        NULL
      </span>
      <Input
        v-else
        class="h-control w-full bg-background"
        :model-value="''"
        :placeholder="_t('Select a field')"
        disabled
      />
    </div>

    <ChoyButton
      class="rm size-control shrink-0 justify-self-end text-muted-foreground opacity-70 hover:bg-destructive/10 hover:text-destructive hover:opacity-100"
      size="icon"
      variant="ghost"
      :aria-label="_t('Remove')"
      @click="onRemove"
    >
      <Trash2 class="size-4" aria-hidden="true" />
    </ChoyButton>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import type { Condition } from '@/web/web/query/types';
import type { WebFieldMetadata, WebModelStore } from '@/web/web/stores/modelStore';
import { useStandaloneField } from '@/web/web/composables/useField';
import { useInjectedFilterEditorBindings } from '@/web/web/composables/search/useFilterEditorBindings';
import { createTranslate } from '@/web/web/i18n';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import Input from '@/web/web/components/vendor/ui/input/Input.vue';
import Select from '@/web/web/components/vendor/ui/select/Select.vue';
import SelectContent from '@/web/web/components/vendor/ui/select/SelectContent.vue';
import SelectItem from '@/web/web/components/vendor/ui/select/SelectItem.vue';
import SelectTrigger from '@/web/web/components/vendor/ui/select/SelectTrigger.vue';
import Combobox from '@/web/web/components/vendor/ui/combobox/Combobox.vue';
import ComboboxAnchor from '@/web/web/components/vendor/ui/combobox/ComboboxAnchor.vue';
import ComboboxContent from '@/web/web/components/vendor/ui/combobox/ComboboxContent.vue';
import ComboboxInput from '@/web/web/components/vendor/ui/combobox/ComboboxInput.vue';
import ComboboxItem from '@/web/web/components/vendor/ui/combobox/ComboboxItem.vue';
import { Trash2 } from 'lucide-vue-next';
import { isConditionComplete } from '@/web/web/query/utils/filter/structures';

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
const incomplete = computed(() => !isConditionComplete(props.condition));

function incompleteRowStyle(): Record<string, string> | undefined {
  if (!incomplete.value) return undefined;
  return {
    boxShadow: 'inset 2px 0 0 color-mix(in oklab, var(--color-warning) 55%, transparent)',
    backgroundColor: 'color-mix(in oklab, var(--color-warning) 6%, transparent)',
  };
}

const operatorOptions = computed(() => getOperatorOptionsForField(props.condition.field));

const fieldOpen = ref(false);
const fieldQuery = ref('');

const selectedFieldLabel = computed(
  () => props.fields.find(f => f.prop === props.condition.field)?.label || '',
);

function fieldDisplayValue(value: string) {
  if (!value) return '';
  return props.fields.find(f => f.prop === value)?.label || value;
}

watch(
  () => [props.condition.field, fieldOpen.value] as const,
  () => {
    if (!fieldOpen.value) {
      fieldQuery.value = selectedFieldLabel.value;
    }
  },
  { immediate: true },
);

const filteredFields = computed(() => {
  const q = fieldQuery.value.trim().toLowerCase();
  const selected = selectedFieldLabel.value.trim().toLowerCase();
  if (!q || (fieldOpen.value && q === selected)) return props.fields;
  return props.fields.filter(
    f => f.label.toLowerCase().includes(q) || f.prop.toLowerCase().includes(q),
  );
});

function onFieldFocus() {
  fieldOpen.value = true;
}

const fieldType = computed(() => metaTypeOf(props.condition.field || ''));
const isRelationValueField = computed(() => fieldType.value === 'manytoone' || fieldType.value === 'manytooneref');

const fieldComponent = computed(() => {
  switch (fieldType.value) {
    case 'char':
    case 'varchar':
    case 'html':
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
    isReadonly: false,
  } as Partial<WebFieldMetadata>;
});

const binding = useStandaloneField({
  value: valueRef,
  meta: fieldMeta.value,
  prop: props.condition.field || 'value',
  env: { isForm: true, isEditMode: true, viewMode: 'edit' },
  record: {},
}) as any;

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
  { immediate: true },
);

function onFieldSelect(val: string | null) {
  if (val == null || val === '') return;
  const ops = getOperatorOptionsForField(val);
  const firstOp = ops.length > 0 ? ops[0].value : undefined;
  props.onUpdateCondition(conditionId.value, { field: val, operator: firstOp, value: undefined });
}

function onFieldCombobox(val: string | null | undefined) {
  if (val == null || val === '') return;
  onFieldSelect(val);
  fieldQuery.value = props.fields.find(f => f.prop === val)?.label || val;
  fieldOpen.value = false;
}

function onOperatorSelect(op: string | null | undefined) {
  if (op == null || op === '') return;
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

function onMultiInput(raw: string) {
  onMultiValuesChange(
    String(raw || '')
      .split(',')
      .map(s => s.trim())
      .filter(Boolean),
  );
}

function onRemove() {
  props.onRemoveCondition(conditionId.value);
}
</script>

<style scoped>
.choy-search-filter__row {
  display: grid;
  grid-template-columns: minmax(8rem, 11rem) minmax(5.75rem, 7.5rem) minmax(0, 1fr) var(--size-control, 2rem);
  align-items: center;
  column-gap: 0.625rem;
}

.choy-search-filter__field,
.choy-search-filter__operator,
.choy-search-filter__value {
  min-width: 0;
}

.choy-search-filter__value-field :deep([data-slot='form-item']),
.choy-search-filter__value-field :deep(.choy-field) {
  margin: 0;
  width: 100%;
}

.choy-search-filter__value-field :deep([data-slot='form-item'] > label),
.choy-search-filter__value-field :deep(.choy-field > label) {
  display: none;
}

.choy-search-filter__value-field :deep([data-slot='form-item'] > div) {
  margin: 0;
  width: 100%;
}

.choy-search-filter__value-field :deep(input),
.choy-search-filter__value-field :deep(textarea),
.choy-search-filter__value-field :deep(button[data-slot='select-trigger']),
.choy-search-filter__value-field :deep([data-slot='combobox-input']) {
  height: var(--choy-control-height, 2rem);
  min-height: var(--choy-control-height, 2rem);
  max-height: var(--choy-control-height, 2rem);
  background-color: var(--color-background);
}
</style>
