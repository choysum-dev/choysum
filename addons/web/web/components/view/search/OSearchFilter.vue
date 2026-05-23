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
        <span class="label">预览 ({{ leafCount }} 条):</span>
        <span class="expr">{{ preview }}</span>
      </div>
      <div class="o-search-filter__actions">
        <el-button @click="$emit('cancel')">取消</el-button>
        <el-button type="primary" @click="$emit('save')">保存</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ConditionGroup, Condition } from '@/web/web/query/types';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import OSearchFilterGroup from './OSearchFilterGroup.vue';
import { valueToPreview } from '@/web/web/query/utils/condition/like';
import { getOperatorLabel } from '@/web/web/query/utils/filter/operators';
import { useFilterEditorBindings } from '@/web/web/composables/search/useFilterEditorBindings';

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

// Reuse metadata helpers from the shared composable.
const { metaTypeOf } = useFilterEditorBindings(props.store as any);

function isGroupNode(n: any): n is ConditionGroup {
  return n && Array.isArray((n as any).children);
}
function nodeId(n: any): string | undefined {
  return (n && (n.tempId || n.id)) as string | undefined;
}

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
  function toSqlOp(op: string): string {
    return /[a-z]/i.test(op) ? op.toUpperCase() : op;
  }

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
    const children: any[] = Array.isArray((g as any).children) ? (g as any).children : [];
    // Treat non-array children as empty.
    const inner = children.map(ch => (isGroupNode(ch) ? expGroup(ch as ConditionGroup) : expCond(ch as Condition))).filter(Boolean);
    if (inner.length === 0) return '(空)';
    if (inner.length === 1) return inner[0]!;
    const joiner = g.logic === 'Or' ? 'OR' : 'AND';
    return `(${inner.join(` ${joiner} `)})`;
  }
  function expCond(c: Condition): string {
    if (!c.field || !c.operator) return '(未完成)';
    const op = String(c.operator);
    const label = getOperatorLabel(op); // Use the user-friendly operator label.
    const pv = toPreviewValue(c.field, c.value);

    if (op === 'is' || op === 'is not') {
      return `${c.field} ${label} ${valueToPreview(c.operator, pv)}`;
    }
    if (op === 'in' || op === 'not in') {
      const arr = Array.isArray(pv) ? pv : pv == null ? [] : [pv];
      const list = `(${arr.map(v => valueToPreview('=', v)).join(', ')})`;
      return `${c.field} ${label} ${list}`;
    }
    return `${c.field} ${label} ${valueToPreview(c.operator, pv)}`;
  }
  function countLeaf(g: ConditionGroup): number {
    let n = 0;
    const children: any[] = Array.isArray((g as any).children) ? (g as any).children : [];
    // Treat non-array children as empty.
    for (const ch of children) {
      n += isGroupNode(ch) ? countLeaf(ch as ConditionGroup) : 1;
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

/* Footer preview and actions. */
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
