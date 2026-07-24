<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="o-search-filter">
    <!-- Root group rendered recursively. -->
    <OSearchFilterGroup
      :group="draft.root"
      :is-root="true"
      :fields="fields"
      :store="store"
      :on-set-logic="onSetLogic"
      :on-add-condition="onAddCondition"
      :on-update-condition="onUpdateCondition"
      :on-remove-condition="onRemoveCondition"
      :on-add-group="onAddGroup"
      :on-remove-group="onRemoveGroup"
    />

    <div class="o-search-filter__footer">
      <div class="o-search-filter__preview">
        <span class="label">{{ _t('Preview (%s):', leafCount) }}</span>
        <span class="expr">{{ preview }}</span>
      </div>
      <div class="o-search-filter__actions">
        <el-button @click="$emit('cancel')">{{ _t('Cancel') }}</el-button>
        <el-button type="primary" @click="$emit('save')">{{ _t('Save') }}</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, provide } from 'vue';
import type { ConditionGroup, Condition } from '@/web/web/query/types';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import OSearchFilterGroup from './OSearchFilterGroup.vue';
import { valueToPreview } from '@/web/web/query/utils/condition/like';
import { getOperatorLabel } from '@/web/web/query/utils/filter/operators';
import { FilterEditorBindingsKey, useFilterEditorBindings } from '@/web/web/composables/search/useFilterEditorBindings';
import { isGroup } from '@/web/web/query/utils/filter/structures';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/search/OSearchFilter' });

interface FieldOption {
  prop: string;
  label: string;
}

// Keep the root draft as a ConditionGroup while preserving the old component contract.
interface DraftFilterCompat {
  baseId?: string;
  root: ConditionGroup;
  dirty?: boolean;
}

const props = defineProps<{
  store: WebModelStore<any>;
  draft: DraftFilterCompat;
  fields: FieldOption[];
}>();

const emit = defineEmits<{
  (e: 'logic-change', logic: 'And' | 'Or', groupId?: string): void;
  (e: 'add-group', parentGroupId?: string): void;
  (e: 'remove-group', groupId: string): void;
  (e: 'add-condition', parentGroupId?: string): void;
  (e: 'remove-condition', id: string): void;
  (e: 'update-condition', id: string, patch: Partial<Condition>): void;
  (e: 'cancel'): void;
  (e: 'save'): void;
}>();

// One bindings instance for the whole editor tree (relation-store cache shared by nested groups).
const bindings = useFilterEditorBindings(props.store as any);
provide(FilterEditorBindingsKey, bindings);
const { metaTypeOf } = bindings;

function onSetLogic(logic: 'And' | 'Or', groupId?: string) {
  emit('logic-change', logic, groupId);
}
function onAddGroup(parentGroupId?: string) {
  emit('add-group', parentGroupId);
}
function onRemoveGroup(groupId: string) {
  emit('remove-group', groupId);
}
function onAddCondition(parentGroupId?: string) {
  emit('add-condition', parentGroupId);
}
function onRemoveCondition(id: string) {
  emit('remove-condition', id);
}
function onUpdateCondition(id: string, patch: Partial<Condition>) {
  emit('update-condition', id, patch);
}

// Recursive preview and leaf counting using SQL-style formatting.
const { preview, leafCount } = (() => {
  // Convert many-to-one objects to ids for preview output.
  function toPreviewValue(field: string, raw: any): any {
    const t = metaTypeOf(field);
    if (t === 'manytoone') {
      const toId = (x: any) => (x && typeof x === 'object' ? (x.Id ?? null) : x);
      return Array.isArray(raw) ? raw.map(toId) : toId(raw);
    }
    return raw;
  }

  function expGroup(g: ConditionGroup): string {
    const children: any[] = Array.isArray(g.children) ? g.children : [];
    const inner = children.map(ch => (isGroup(ch) ? expGroup(ch) : expCond(ch as Condition))).filter(Boolean);
    if (inner.length === 0) return _t('(empty)');
    if (inner.length === 1) return inner[0]!;
    const joiner = g.logic === 'Or' ? 'OR' : 'AND';
    return `(${inner.join(` ${joiner} `)})`;
  }
  function expCond(c: Condition): string {
    if (!c.field || !c.operator) return _t('(incomplete)');
    const op = String(c.operator);
    const label = getOperatorLabel(op);
    const pv = toPreviewValue(c.field, c.value);
    const previewOpts = { fieldType: metaTypeOf(c.field) };

    if (op === 'is' || op === 'is not') {
      return `${c.field} ${label} ${valueToPreview(c.operator, pv, previewOpts)}`;
    }
    if (op === 'in' || op === 'not in') {
      const arr = Array.isArray(pv) ? pv : pv == null ? [] : [pv];
      const list = `(${arr.map(v => valueToPreview('=', v, previewOpts)).join(', ')})`;
      return `${c.field} ${label} ${list}`;
    }
    return `${c.field} ${label} ${valueToPreview(c.operator, pv, previewOpts)}`;
  }
  function countLeaf(g: ConditionGroup): number {
    let n = 0;
    const children: any[] = Array.isArray(g.children) ? g.children : [];
    for (const ch of children) {
      n += isGroup(ch) ? countLeaf(ch) : 1;
    }
    return n;
  }
  return {
    preview: computed(() => expGroup(props.draft.root)),
    leafCount: computed(() => countLeaf(props.draft.root)),
  };
})();
</script>

<style scoped lang="scss">
.o-search-filter {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.o-search-filter__footer {
  display: flex;
  flex-direction: column;
  gap: 12px;
  border-top: 1px solid var(--el-border-color-light);
  padding-top: 12px;
}
.o-search-filter__preview {
  font-size: 12px;
  display: flex;
  gap: 6px;
  .label {
    color: var(--el-text-color-secondary);
  }
  .expr {
    word-break: break-all;
  }
}
.o-search-filter__actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
