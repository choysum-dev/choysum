<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div :class="['osf-group', group.logic === 'And' ? 'osf-group--and rounded-md border border-border border-l-[3px] border-l-success p-3 mb-3 transition-colors hover:bg-success-subtle' : 'osf-group--or rounded-md border border-border border-l-[3px] border-l-warning p-3 mb-3 transition-colors hover:bg-warning-subtle']">
    <div class="osf-group__header flex items-center justify-between">
        <div class="osf-group__relation flex items-center gap-2">
        <span :class="group.logic === 'And' ? 'text-success' : 'text-warning'">{{ _t('Group relation') }}</span>
        <div class="rg" role="group" :aria-label="_t('Group relation')">
          <button
            type="button"
            class="radio"
            data-value="And"
            :aria-pressed="group.logic === 'And'"
            @click="onLogicChange('And')"
          >
            AND
          </button>
          <button
            type="button"
            class="radio"
            data-value="Or"
            :aria-pressed="group.logic === 'Or'"
            @click="onLogicChange('Or')"
          >
            OR
          </button>
        </div>
      </div>
      <div class="osf-group__ops flex gap-2">
        <ChoyButton class="btn" size="sm" @click="onAddCondition(group.tempId || group.id)">+ {{ _t('Add condition') }}</ChoyButton>
        <ChoyButton class="btn" size="sm" @click="onAddGroup(group.tempId || group.id)">+ {{ _t('Add group') }}</ChoyButton>
        <ChoyButton v-if="!isRoot" class="btn" size="sm" variant="destructive" @click="onRemoveGroup(group.tempId || group.id)">{{ _t('Remove group') }}</ChoyButton>
      </div>
    </div>

    <hr border-style="none" class="my-2" />

    <div class="osf-group__children flex flex-col gap-2.5 ps-1.5">
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

