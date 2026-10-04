<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyDropdownMenu>
    <ChoyDropdownMenuTrigger as-child>
      <ChoyButton
        variant="ghost"
        size="icon"
        type="button"
        :aria-label="tLayout('layout.header.theme')"
        data-testid="choy-shell-theme"
      >
        <Moon v-if="resolvedDark" class="size-4" aria-hidden="true" />
        <Sun v-else class="size-4" aria-hidden="true" />
      </ChoyButton>
    </ChoyDropdownMenuTrigger>
    <ChoyDropdownMenuContent align="end" class="min-w-[10rem]">
      <ChoyDropdownMenuItem
        v-for="item in themeItems"
        :key="item.mode"
        :data-testid="`choy-shell-theme-${item.mode}`"
        class="gap-2"
        @select="applyTheme(item.mode)"
      >
        <Check class="size-4" :class="mode === item.mode ? 'opacity-100' : 'opacity-0'" aria-hidden="true" />
        {{ item.label }}
      </ChoyDropdownMenuItem>
    </ChoyDropdownMenuContent>
  </ChoyDropdownMenu>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { Check, Moon, Sun } from 'lucide-vue-next';
import ChoyButton from './ChoyButton.vue';
import {
  DropdownMenu as ChoyDropdownMenu,
  DropdownMenuContent as ChoyDropdownMenuContent,
  DropdownMenuItem as ChoyDropdownMenuItem,
  DropdownMenuTrigger as ChoyDropdownMenuTrigger,
} from '../vendor/ui/dropdown-menu';
import {
  applyChoyThemePreference,
  readChoyThemePreference,
  resolveChoyThemePreference,
  type ChoyThemeMode,
} from '../../composables/applyChoyThemePreference';

defineOptions({ name: 'ChoyShellThemeToggle' });

let tLayout: (key: string) => string = (key) => key;
try {
  const i18n = useI18n({ useScope: 'global' });
  tLayout = (key) => String(i18n.t(key));
} catch {
  tLayout = (key) => key;
}

const mode = ref<ChoyThemeMode>(readChoyThemePreference().theme || 'light');

const resolvedDark = computed(
  () => resolveChoyThemePreference({ theme: mode.value }).dark,
);

const themeItems = computed(() => [
  { mode: 'light' as const, label: tLayout('layout.header.lightMode') },
  { mode: 'dark' as const, label: tLayout('layout.header.darkMode') },
  { mode: 'auto' as const, label: tLayout('layout.header.autoMode') },
]);

function applyTheme(next: ChoyThemeMode) {
  mode.value = next;
  applyChoyThemePreference({ theme: next });
}
</script>
