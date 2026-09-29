<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div class="osf-group" :class="group.logic === 'And' ? 'osf-group--and' : 'osf-group--or'">
    <div class="osf-group__header">
        <div class="osf-group__relation">
        <span class="osf-group__relation-label">{{ _t('Group relation') }}</span>
        <div class="rg">
          <div class="radio" data-value="And">AND</div>
          <div class="radio" data-value="Or">OR</div>
          <button type="button" class="to-or" @click="onLogicChange('Or')">OR</button>
        </div>
      </div>
      <div class="osf-group__ops">
        <ChoyButton class="btn" size="sm" @click="onAddCondition(group.tempId || group.id)">+ {{ _t('Add condition') }}</ChoyButton>
        <ChoyButton class="btn" size="sm" @click="onAddGroup(group.tempId || group.id)">+ {{ _t('Add group') }}</ChoyButton>
        <ChoyButton v-if="!isRoot" class="btn" size="sm" variant="destructive" @click="onRemoveGroup(group.tempId || group.id)">{{ _t('Remove group') }}</ChoyButton>
      </div>
    </div>

    <hr border-style="none" class="osf-group__divider" />

    <div class="osf-group__children">
      <template v-for="ch in group.children" :key="nodeKey(ch)">
        <SearchFilterGroup
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
        <SearchFilterCondition
          v-else
          :condition="ch as any"
          :fields="fields"
          :store="store"
          :on-update-condition="onUpdateCondition"
          :on-remove-condition="onRemoveCondition"
        />
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConditionGroup, Condition } from '@/web/web/query/types';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import SearchFilterCondition from './SearchFilterCondition.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/view/search/SearchFilterGroup' });

defineOptions({ name: 'SearchFilterGroup' });

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
</script>

<style scoped>
.osf-group {
  border: 1px solid var(--el-border-color-lighter);border-radius: 6px;padding: 12px;margin-bottom: 12px;border-left-width: 3px;border-left-style: solid;transition:
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
  display: flex;align-items: center;justify-content: space-between;
}
.osf-group__relation {
  display: flex;align-items: center;gap: 8px;
}
.osf-group--and .osf-group__relation-label {
  color: var(--el-color-success);
}
.osf-group--or .osf-group__relation-label {
  color: var(--el-color-warning);
}
.osf-group__ops {
  display: flex;gap: 8px;
}
.osf-group__divider {
  margin: 8px 0;
}
.osf-group__children {
  display: flex;flex-direction: column;gap: 10px;padding-left: 6px;
}
</style>
