<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDropdownMenu v-if="rows.length">
    <ChoyDropdownMenuTrigger as-child>
      <ChoyButton
        variant="ghost"
        size="icon"
        type="button"
        :aria-label="_t('Languages')"
        data-testid="choy-shell-locale"
      >
        <Languages class="size-4" aria-hidden="true" />
      </ChoyButton>
    </ChoyDropdownMenuTrigger>
    <ChoyDropdownMenuContent align="end" class="max-h-72 min-w-[10rem] overflow-y-auto">
      <ChoyDropdownMenuItem
        v-for="row in rows"
        :key="row.Code"
        role="menuitemradio"
        :aria-checked="row.Code === currentCode"
        :data-testid="`choy-shell-locale-${row.Code}`"
        class="gap-2"
        @select="onSelect(row.Code)"
      >
        <Check
          class="size-4"
          :class="row.Code === currentCode ? 'opacity-100' : 'opacity-0'"
          aria-hidden="true"
        />
        {{ row.Name || row.Code }}
      </ChoyDropdownMenuItem>
    </ChoyDropdownMenuContent>
  </ChoyDropdownMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { createTranslate } from '@/web/web/i18n';
import { Check, Languages } from 'lucide-vue-next';
import ChoyButton from './ChoyButton.vue';
import {
  DropdownMenu as ChoyDropdownMenu,
  DropdownMenuContent as ChoyDropdownMenuContent,
  DropdownMenuItem as ChoyDropdownMenuItem,
  DropdownMenuTrigger as ChoyDropdownMenuTrigger,
} from '../vendor/ui/dropdown-menu';
import { afterLangChange, useI18nStore } from '../../stores/i18nStore';
import { shellLanguageSelect } from './shellLanguageSelect';
import { DEFAULT_ACTIVE_LANGS } from '../../stores/i18nStore/activeLangs';
import type { ActiveLanguageRow } from '../../stores/i18nStore/types';

defineOptions({ name: 'ChoyShellLanguageMenu' });

const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyShellLanguageMenu' });

let i18nStore: ReturnType<typeof useI18nStore> | null = null;
try {
  i18nStore = useI18nStore();
} catch {
  i18nStore = null;
}

function resolveI18nStore(): ReturnType<typeof useI18nStore> | null {
  if (i18nStore) {
    return i18nStore;
  }
  try {
    return useI18nStore();
  } catch {
    return null;
  }
}

const rows = computed((): ActiveLanguageRow[] => {
  const fromStore = resolveI18nStore()?.activeLanguages;
  if (Array.isArray(fromStore) && fromStore.length) {
    return fromStore;
  }
  return DEFAULT_ACTIVE_LANGS.map(code => ({ Code: code, Name: code }));
});

const currentCode = computed(() => resolveI18nStore()?.activeLanguage.code || 'en_US');

async function onSelect(code: string) {
  const store = resolveI18nStore();
  if (!store) {
    return;
  }
  await shellLanguageSelect(code, c => store.setLang(c), afterLangChange);
}
</script>