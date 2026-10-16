<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Static class tokens for kit Tailwind scan. :class ternaries are not
    extracted as candidates; keep any dynamic-only utilities listed here.
  -->
  <span
    class="hidden border-l-2 border-l-success border-l-warning border-border/60 bg-success/15 bg-warning/10 bg-warning/20 text-success text-warning"
    hidden
    aria-hidden="true"
  />
  <div
    :class="[
      'osf-group min-w-0',
      group.logic === 'And' ? 'osf-group--and' : 'osf-group--or',
      isRoot
        ? 'overflow-hidden rounded-lg border border-border'
        : [
            'mt-2 rounded-md border border-border/60 border-l-2 ps-3',
            group.logic === 'Or' ? 'border-l-warning bg-warning/10' : 'border-l-success bg-success/15',
          ],
    ]"
    :style="isRoot ? rootShellStyle() : undefined"
    :data-logic="group.logic"
  >
    <div
      :class="[
        'osf-group__header flex flex-wrap items-center justify-between gap-2',
        isRoot ? 'border-b border-border px-3 py-2' : 'py-2',
      ]"
      :style="isRoot ? rootHeaderStyle() : undefined"
    >
      <div class="osf-group__relation flex min-w-0 items-center gap-2">
        <span
          v-if="isRoot"
          class="text-xs font-medium text-muted-foreground"
        >
          {{ _t('Group relation') }}
        </span>
        <ToggleGroup
          class="rg osf-group__logic rounded-md bg-muted p-0.5"
          type="single"
          variant="default"
          size="sm"
          :model-value="group.logic"
          :data-logic="group.logic"
          :aria-label="_t('Group relation')"
          @update:model-value="onLogicToggle"
        >
          <ToggleGroupItem
            class="osf-group__logic-and min-w-11 px-2.5 text-xs"
            :class="group.logic === 'And' ? 'font-semibold shadow-sm' : 'text-muted-foreground'"
            :style="logicItemStyle('And')"
            :data-logic-active="group.logic === 'And' ? 'true' : 'false'"
            value="And"
          >
            AND
          </ToggleGroupItem>
          <ToggleGroupItem
            class="osf-group__logic-or min-w-11 px-2.5 text-xs"
            :class="group.logic === 'Or' ? 'font-semibold shadow-sm' : 'text-muted-foreground'"
            :style="logicItemStyle('Or')"
            :data-logic-active="group.logic === 'Or' ? 'true' : 'false'"
            value="Or"
          >
            OR
          </ToggleGroupItem>
        </ToggleGroup>
      </div>

      <div class="osf-group__ops flex flex-wrap items-center gap-0.5">
        <ChoyButton
          class="btn text-muted-foreground hover:text-foreground"
          :class="isRoot ? 'h-control-sm px-2 text-xs' : 'size-control'"
          :size="isRoot ? 'sm' : 'icon'"
          variant="ghost"
          :aria-label="_t('Add condition')"
          :title="isRoot ? undefined : _t('Add condition')"
          @click="onAddCondition(groupId)"
        >
          <Plus class="size-3.5" aria-hidden="true" />
          <span v-if="isRoot">{{ _t('Add condition') }}</span>
        </ChoyButton>
        <ChoyButton
          class="btn text-muted-foreground hover:text-foreground"
          :class="isRoot ? 'h-control-sm px-2 text-xs' : 'size-control'"
          :size="isRoot ? 'sm' : 'icon'"
          variant="ghost"
          :aria-label="_t('Add group')"
          :title="isRoot ? undefined : _t('Add group')"
          @click="onAddGroup(groupId)"
        >
          <Layers class="size-3.5" aria-hidden="true" />
          <span v-if="isRoot">{{ _t('Add group') }}</span>
        </ChoyButton>
        <ChoyButton
          v-if="!isRoot"
          class="btn size-control text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
          size="icon"
          variant="ghost"
          :aria-label="_t('Remove group')"
          :title="_t('Remove group')"
          @click="onRemoveGroup(groupId)"
        >
          <Trash2 class="size-3.5" aria-hidden="true" />
        </ChoyButton>
      </div>
    </div>

    <div
      :class="[
        'osf-group__children flex flex-col',
        isRoot ? 'gap-3 p-4' : 'gap-2 px-1 pb-3 pt-2',
      ]"
    >
      <p
        v-if="!group.children.length"
        class="osf-group__empty py-2 text-xs text-muted-foreground"
      >
        {{ _t('Add a condition to start filtering') }}
      </p>
      <template v-for="(ch, idx) in group.children" :key="nodeKey(ch)">
        <div
          v-if="idx > 0"
          class="osf-group__joiner flex items-center gap-2 py-1"
          aria-hidden="true"
        >
          <span class="h-px min-w-4 flex-1 bg-border/70" />
          <span
            class="osf-group__joiner-label shrink-0 rounded px-1.5 py-px text-[0.65rem] font-semibold leading-4 tracking-wide"
            :class="group.logic === 'Or' ? 'bg-warning/20 text-warning' : 'bg-success/15 text-success'"
            :data-logic="group.logic"
          >
            {{ group.logic === 'Or' ? 'OR' : 'AND' }}
          </span>
          <span class="h-px min-w-4 flex-1 bg-border/70" />
        </div>
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
import { computed } from 'vue';
import type { ConditionGroup, Condition } from '@/web/web/query/types';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import SearchFilterCondition from './SearchFilterCondition.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ToggleGroup from '@/web/web/components/vendor/ui/toggle-group/ToggleGroup.vue';
import ToggleGroupItem from '@/web/web/components/vendor/ui/toggle-group/ToggleGroupItem.vue';
import { Layers, Plus, Trash2 } from 'lucide-vue-next';
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

const {
  onSetLogic,
  onAddGroup,
  onRemoveGroup,
  onAddCondition,
  onUpdateCondition,
  onRemoveCondition,
} = props;

const groupId = computed(() => props.group.tempId || props.group.id);

function logicTintToken(): string {
  return props.group.logic === 'Or' ? 'var(--color-warning)' : 'var(--color-success)';
}

// Root shell stays lighter than nested success/warning fills.
function rootShellStyle(): Record<string, string> {
  return {
    backgroundColor: 'color-mix(in oklab, ' + logicTintToken() + ' 7%, var(--color-card))',
  };
}

// Root header is a step stronger than the shell, still below nested fill.
function rootHeaderStyle(): Record<string, string> {
  return {
    backgroundColor: 'color-mix(in oklab, ' + logicTintToken() + ' 12%, transparent)',
  };
}

// Inline token colors beat Toggle selected accent specificity.
function logicItemStyle(value: Logic): Record<string, string> | undefined {
  if (props.group.logic !== value) return undefined;
  return {
    backgroundColor: value === 'Or' ? 'var(--color-warning)' : 'var(--color-success)',
    color: 'var(--color-primary-foreground)',
  };
}

function onLogicToggle(val: string | string[] | undefined | null) {
  const next = Array.isArray(val) ? val[0] : val;
  if (next !== 'And' && next !== 'Or') return;
  // Ignore clear/deselect so the toggle cannot leave the group without a logic.
  if (next === props.group.logic) return;
  onSetLogic(next, groupId.value);
}
function isDraftGroup(n: any): n is GroupLike {
  return n && Array.isArray((n as any).children);
}
function nodeKey(n: any) {
  return (n.tempId || n.id) as string;
}
</script>
