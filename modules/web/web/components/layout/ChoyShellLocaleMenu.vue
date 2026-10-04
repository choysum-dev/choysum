<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDropdownMenu v-if="codes.length">
    <ChoyDropdownMenuTrigger as-child>
      <ChoyButton
        variant="ghost"
        size="sm"
        type="button"
        :aria-label="tLayout('layout.header.languages')"
        data-testid="choy-shell-locale"
      >
        {{ currentLabel }}
      </ChoyButton>
    </ChoyDropdownMenuTrigger>
    <ChoyDropdownMenuContent align="end" class="min-w-[10rem]">
      <ChoyDropdownMenuItem
        v-for="code in codes"
        :key="code"
        @select="onSelect(code)"
      >
        {{ localeName(code) }}
      </ChoyDropdownMenuItem>
    </ChoyDropdownMenuContent>
  </ChoyDropdownMenu>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import ChoyButton from './ChoyButton.vue';
import {
  DropdownMenu as ChoyDropdownMenu,
  DropdownMenuContent as ChoyDropdownMenuContent,
  DropdownMenuItem as ChoyDropdownMenuItem,
  DropdownMenuTrigger as ChoyDropdownMenuTrigger,
} from '../vendor/ui/dropdown-menu';
import { useI18nStore } from '../../stores/i18nStore';
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
  const fromStore = i18nStore?.supportedLocales;
  if (Array.isArray(fromStore) && fromStore.length) return fromStore as SupportedLocale[];
  return Object.keys(SUPPORTED_LOCALES) as SupportedLocale[];
});

const currentLabel = computed(() => {
  const code = (i18nStore?.localeCode as SupportedLocale | null) || 'en';
  return localeName(code);
});

function localeName(code: string): string {
  return SUPPORTED_LOCALES[code as SupportedLocale]?.name || code;
}

function onSelect(code: SupportedLocale) {
  void i18nStore?.setUiKey(code);
}
</script>
