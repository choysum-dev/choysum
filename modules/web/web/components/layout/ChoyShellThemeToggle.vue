<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyButton
    variant="ghost"
    size="icon"
    type="button"
    :aria-label="label"
    data-testid="choy-shell-theme"
    @click="cycleTheme"
  >
    <Sun v-if="mode === 'light'" class="size-4" aria-hidden="true" />
    <Moon v-else-if="mode === 'dark'" class="size-4" aria-hidden="true" />
    <Monitor v-else class="size-4" aria-hidden="true" />
  </ChoyButton>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { Monitor, Moon, Sun } from 'lucide-vue-next';
import ChoyButton from './ChoyButton.vue';
import {
  applyChoyThemePreference,
  nextChoyThemeMode,
  readChoyThemePreference,
  type ChoyThemeMode,
} from '../../composables/applyChoyThemePreference';

defineOptions({ name: 'ChoyShellThemeToggle' });

const mode = ref<ChoyThemeMode>(readChoyThemePreference().theme || 'light');

const label = computed(() => {
  if (mode.value === 'dark') return 'Dark';
  if (mode.value === 'auto') return 'Auto';
  return 'Light';
});

function cycleTheme() {
  mode.value = nextChoyThemeMode(mode.value);
  applyChoyThemePreference({ theme: mode.value });
}
</script>
