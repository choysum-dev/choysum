<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="choy-switch-company__panel flex w-full flex-col gap-1 p-1"
    data-testid="company-switch-panel-body"
  >
    <div
      class="flex items-center gap-2 px-1 pb-1"
      data-testid="company-switch-toolbar"
    >
      <ChoyCheckbox
        class="size-4 shrink-0"
        :model-value="selectAllState"
        :disabled="disabled || branchEditsDisabled || allowedCount < 1"
        :aria-label="_t('Select all')"
        data-testid="company-switch-select-all"
        @update:model-value="emit('toggle-select-all')"
      />
      <ChoyInput
        v-model="searchFilter"
        :placeholder="_t('Search companies')"
        class="h-8 min-w-0 flex-1"
        data-testid="company-switch-search"
      />
    </div>

    <div class="bg-border mx-1 h-px" role="separator" aria-hidden="true" />

    <div
      class="choy-switch-company__list flex max-h-64 flex-col overflow-y-auto"
      data-testid="company-enabled-select"
      :aria-label="_t('Companies')"
    >
      <div
        v-for="entry in entries"
        :key="entry.id"
        class="flex items-center gap-2 rounded-sm px-2 py-1.5 text-sm"
        :class="
          entry.id === activeCompanyId
            ? 'bg-accent text-accent-foreground'
            : 'hover:bg-accent hover:text-accent-foreground'
        "
        :style="{ paddingInlineStart: `${0.5 + entry.level * 0.75}rem` }"
        data-testid="company-switch-item"
        :data-company-id="entry.id"
      >
        <ChoyCheckbox
          class="size-4 shrink-0"
          :model-value="entry.checked"
          :disabled="disabled || branchEditsDisabled || entry.toggleDisabled"
          :aria-label="`${_t('Toggle company')}: ${entry.name || entry.id}`"
          data-testid="company-switch-toggle"
          @click.stop
          @update:model-value="emit('toggle-branch', entry.id)"
        />
        <button
          type="button"
          class="flex min-w-0 flex-1 items-center gap-2 rounded-sm text-start outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
          data-testid="company-switch-loginto"
          :disabled="disabled"
          :aria-current="entry.id === activeCompanyId ? 'true' : undefined"
          @click.stop.prevent="emit('loginto', entry.id)"
        >
          <span class="min-w-0 flex-1 truncate">{{ entry.name || entry.id }}</span>
          <Check
            v-if="entry.id === activeCompanyId"
            class="text-foreground size-4 shrink-0"
            aria-hidden="true"
          />
        </button>
      </div>
    </div>

    <template v-if="dirty && !disabled">
      <div class="bg-border mx-1 my-1 h-px" role="separator" aria-hidden="true" />
      <div
        class="choy-switch-company__actions flex justify-end gap-1.5 px-1 pb-0.5"
        data-testid="company-switch-actions"
      >
        <ChoyButton
          size="sm"
          variant="ghost"
          :disabled="disabled"
          data-testid="company-switch-reset"
          @click.stop="emit('reset')"
        >
          {{ _t('Reset') }}
        </ChoyButton>
        <ChoyButton
          size="sm"
          :disabled="disabled"
          data-testid="company-switch-confirm"
          @click.stop="emit('confirm')"
        >
          {{ _t('Confirm') }}
        </ChoyButton>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { Check } from 'lucide-vue-next';
import { ChoyButton, ChoyCheckbox, ChoyInput } from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'SwitchCompanyPanel' });

export type SwitchCompanyPanelEntry = {
  id: string;
  level: number;
  name: string;
  checked: boolean | 'indeterminate';
  toggleDisabled: boolean;
};

withDefaults(
  defineProps<{
    selectAllState: boolean | 'indeterminate';
    allowedCount: number;
    entries: SwitchCompanyPanelEntry[];
    activeCompanyId: string;
    dirty: boolean;
    disabled?: boolean;
    /** When false, branch checkboxes stay off until ParentId hierarchy is loaded. */
    branchEditsDisabled?: boolean;
  }>(),
  { disabled: false, branchEditsDisabled: false },
);

const emit = defineEmits<{
  'toggle-select-all': [];
  'toggle-branch': [companyId: string];
  loginto: [companyId: string];
  reset: [];
  confirm: [];
}>();

const searchFilter = defineModel<string>('searchFilter', { default: '' });

const { _t } = createTranslate('auth', { scope: 'web/components/layout/SwitchCompany' });
</script>
