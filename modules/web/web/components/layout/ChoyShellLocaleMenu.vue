<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDropdownMenu v-if="codes.length">
    <ChoyDropdownMenuTrigger as-child>
      <ChoyButton
        variant="ghost"
        size="icon"
        type="button"
        :aria-label="tLayout('layout.header.languages')"
        data-testid="choy-shell-locale"
      >
        <Languages class="size-4" aria-hidden="true" />
      </ChoyButton>
    </ChoyDropdownMenuTrigger>
    <ChoyDropdownMenuContent align="end" class="max-h-72 min-w-[10rem] overflow-y-auto">
      <ChoyDropdownMenuItem
        v-for="code in codes"
        :key="code"
        role="menuitemradio"
        :aria-checked="code === currentCode"
        :data-testid="`choy-shell-locale-${code}`"
        class="gap-2"
        @select="onSelect(code)"
      >
        <Check class="size-4" :class="code === currentCode ? 'opacity-100' : 'opacity-0'" aria-hidden="true" />
        {{ localeName(code) }}
      </ChoyDropdownMenuItem>
    </ChoyDropdownMenuContent>
  </ChoyDropdownMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { Check, Languages } from 'lucide-vue-next';
import ChoyButton from './ChoyButton.vue';
import {
  DropdownMenu as ChoyDropdownMenu,
  DropdownMenuContent as ChoyDropdownMenuContent,
  DropdownMenuItem as ChoyDropdownMenuItem,
  DropdownMenuTrigger as ChoyDropdownMenuTrigger,
} from '../vendor/ui/dropdown-menu';
import { useI18nStore } from '../../stores/i18nStore';
import { DEFAULT_ACTIVE_UI_KEYS } from '../../stores/i18nStore/activeUiKeys';
import { SUPPORTED_LOCALES } from '../../stores/i18nStore/locales';
import { type SupportedLocale } from '../../stores/i18nStore/types';

defineOptions({ name: 'ChoyShellLocaleMenu' });

let tLayout: (key: string) => string = (key) => key;
try {
  const i18n = useI18n({ useScope: 'global' });
  tLayout = (key) => String(i18n.t(key));
} catch {
  tLayout = (key) => key;
}

let i18nStore: ReturnType<typeof useI18nStore> | null = null;
try {
  i18nStore = useI18nStore();
} catch {
  i18nStore = null;
}

const codes = computed(() => {
  const fromStore = i18nStore?.activeUiKeys;
  if (Array.isArray(fromStore) && fromStore.length) return fromStore as SupportedLocale[];
  return [...DEFAULT_ACTIVE_UI_KEYS] as SupportedLocale[];
});

const currentCode = computed(
  () => (i18nStore?.localeCode as SupportedLocale | null) || 'en',
);

function localeName(code: string): string {
  return SUPPORTED_LOCALES[code as SupportedLocale]?.name || code;
}

function onSelect(code: SupportedLocale) {
  void i18nStore?.setUiKey(code);
}
</script>
