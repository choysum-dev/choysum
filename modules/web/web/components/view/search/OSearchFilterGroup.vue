<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="osf-group" :class="group.logic === 'And' ? 'osf-group--and' : 'osf-group--or'">
    <div class="osf-group__header">
      <div class="osf-group__relation">
        <span class="osf-group__relation-label">{{ _t('Group relation') }}</span>
        <el-radio-group size="small" :model-value="group.logic" @update:model-value="onLogicChange">
          <el-radio label="And">AND</el-radio>
          <el-radio label="Or">OR</el-radio>
        </el-radio-group>
      </div>
      <div class="osf-group__ops">
        <el-button size="small" @click="onAddCondition(group.tempId || group.id)">+ {{ _t('Add condition') }}</el-button>
        <el-button size="small" @click="onAddGroup(group.tempId || group.id)">+ {{ _t('Add group') }}</el-button>
        <el-button v-if="!isRoot" size="small" type="danger" text @click="onRemoveGroup(group.tempId || group.id)">{{ _t('Remove group') }}</el-button>
      </div>
    </div>

    <el-divider border-style="none" class="osf-group__divider" />

    <div class="osf-group__children">
      <template v-for="ch in group.children" :key="nodeKey(ch)">
        <!-- Nested subgroup. -->
        <OSearchFilterGroup
          v-if="isDraftGroup(ch)"
          :group="ch as any"
          :is-root="false"
          :fields="fields"
          :store="store"
          :on-set-logic="onSetLogic"
          :on-add-group="onAddGroup"
          :on-remove-group="onRemoveGroup"
          :on-add-condition="onAddCondition"
          :on-update-condition="onUpdateCondition"
          :on-remove-condition="onRemoveCondition"
        />

        <!-- Condition row. -->
        <div v-else class="o-search-filter__row">
          <!-- Field selector. -->
          <el-select
            class="w-field"
            :placeholder="_t('Field')"
            :model-value="(ch as any).field"
            @update:model-value="
              (val: string) => {
                const ops = getOperatorOptions(val);
                const firstOp = ops.length > 0 ? ops[0].value : undefined;
                onUpdateCondition((ch as any).tempId || (ch as any).id, { field: val, operator: firstOp, value: undefined });
              }
            "
            filterable
            clearable
          >
            <el-option v-for="f in fields" :key="f.prop" :label="f.label" :value="f.prop" />
          </el-select>

          <!-- Operator selector derived from the field metadata. -->
          <el-select
            class="w-operator"
            :placeholder="_t('Operator')"
            :disabled="!(ch as any).field"
            :model-value="(ch as any).operator"
            @update:model-value="(val: string) => onOperatorChange(ch as any, val)"
            clearable
          >
            <el-option v-for="op in getOperatorOptions((ch as any).field)" :key="op.value" :label="op.label" :value="op.value" />
          </el-select>

          <!-- Value editor selected by metadata type; NULL operators do not need a value. -->
          <component
            v-if="(ch as any).field && requiresValue((ch as any).operator)"
            :is="resolveOField(metaTypeOf((ch as any).field))"
            class="w-value"
            :binding="bindingForCondition(ch as any)"
            v-bind="extraPropsFor((ch as any).field)"
            :label="''"
            :rules="[]"
            :placeholder="valuePlaceholder(metaTypeOf((ch as any).field))"
            :formItemProps="{ labelWidth: 0, style: { margin: 0, padding: 0 } }"
          />
          <span v-else-if="(ch as any).field && isNullOperator((ch as any).operator)" class="w-value o-null-flag">NULL</span>
          <el-input v-else class="w-value" :placeholder="_t('Select a field')" disabled />

          <el-button type="danger" text size="small" @click="onRemoveCondition((ch as any).tempId || (ch as any).id)">{{ _t('Remove') }}</el-button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConditionGroup, Condition } from '@/web/web/query/types';
import { computed } from 'vue';
import { useStandaloneField } from '@/web/web/composables/useField';
import type { WebFieldMetadata } from '@/web/web/stores/modelStore';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useFilterEditorBindings } from '@/web/web/composables/search/useFilterEditorBindings';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/search/OSearchFilterGroup' });

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

defineOptions({ name: 'OSearchFilterGroup' });

interface FieldOption {
  prop: string;
  label: string;
}
type Logic = 'And' | 'Or';

type CondLike = Condition & { tempId?: string };
type GroupLike = ConditionGroup & { tempId?: string; children: Array<CondLike | GroupLike> };

const props = defineProps<{
  group: GroupLike;
  isRoot?: boolean;
  fields: FieldOption[];
  store: WebModelStore<any>;

  onSetLogic: (logic: Logic, groupId?: string) => void;
  onAddGroup: (parentGroupId?: string) => void;
  onRemoveGroup: (groupId: string) => void;
  onAddCondition: (parentGroupId?: string) => void;
  onUpdateCondition: (id: string, patch: Partial<CondLike>) => void;
  onRemoveCondition: (id: string) => void;
}>();

const { group, isRoot = false, fields, store, onSetLogic, onAddGroup, onRemoveGroup, onAddCondition, onUpdateCondition, onRemoveCondition } = props;

function onLogicChange(val: Logic) {
  onSetLogic(val, (group as any).tempId || group.id);
}
function isDraftGroup(n: any): n is GroupLike {
  return n && Array.isArray((n as any).children);
}
function nodeKey(n: any) {
  return (n.tempId || n.id) as string;
}

/* Shared metadata, relation, operator, and value helpers. */
const { metaTypeOf, relationStoreOf, getOperatorOptionsForField, isNullOperator, requiresValue, defaultValueFor } = useFilterEditorBindings(store as any);
// Keep the getOperatorOptions name for template compatibility.
const getOperatorOptions = getOperatorOptionsForField;

/* Handle operator changes in one place. */
function onOperatorChange(ch: CondLike, op: string) {
  const id = (ch as any).tempId || ch.id;
  onUpdateCondition(id, { operator: op });

  if (isNullOperator(op)) {
    onUpdateCondition(id, { value: null });
    return;
  }

  if (requiresValue(op) && (ch.value === undefined || ch.value === null)) {
    const t = metaTypeOf(ch.field || '');
    const dv = defaultValueFor(t);
    if (dv !== undefined) {
      onUpdateCondition(id, { value: dv });
    }
  }
}

/* Pick the O*Field component from the original metadata type. */
function resolveOField(t: string) {
  switch (t) {
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
    default:
      return OVarCharField;
  }
}

/* Add type-specific props, including many-to-one object/id conversion. */
function extraPropsFor(field?: string) {
  const t = metaTypeOf(field || '');
  if (t === 'manytoone') {
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
  }
  return {};
}

/* Placeholder text keyed by the original metadata type. */
function valuePlaceholder(t: string) {
  switch (t) {
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
    default:
      return _t('Value');
  }
}

/* Independent binding for each condition value. */
function bindingForCondition(ch: CondLike) {
  const id = (ch as any).tempId || ch.id;
  const vRef = computed({
    get: () => ch.value,
    set: v => onUpdateCondition(id, { value: v }),
  });

  const t = metaTypeOf(ch.field || '');
  const meta = { type: t as any } as Partial<WebFieldMetadata>;

  const binding = useStandaloneField({
    value: vRef,
    meta,
    prop: ch.field || 'value',
    env: { isForm: true, isEditMode: true, viewMode: 'edit' },
    record: {},
  }) as any;

  if (t === 'manytoone') {
    binding.relationStore = relationStoreOf(ch.field || '');
  }
  return binding;
}
</script>

<style scoped lang="scss">
.osf-group {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
  border-left-width: 3px;
  border-left-style: solid;
  transition:
    background-color 0.15s ease,
    border-color 0.15s ease;
}
.osf-group--and {
  border-left-color: var(--el-color-success);
}
.osf-group--and:hover {
  background-color: var(--el-color-success-light-9);
}
.osf-group--or {
  border-left-color: var(--el-color-warning);
}
.osf-group--or:hover {
  background-color: var(--el-color-warning-light-9);
}

.osf-group__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.osf-group__relation {
  display: flex;
  align-items: center;
  gap: 8px;
}
.osf-group--and .osf-group__relation-label {
  color: var(--el-color-success);
}
.osf-group--or .osf-group__relation-label {
  color: var(--el-color-warning);
}

.osf-group__ops {
  display: flex;
  gap: 8px;
}
.osf-group__divider {
  margin: 8px 0;
}
.osf-group__children {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-left: 6px;
}

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
