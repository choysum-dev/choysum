<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="_t('Home')" padding width="wide">
    <template #title-actions>
      <ChoyButton type="button" variant="outline" size="sm" @click="refresh">
        {{ _t('Refresh') }}
      </ChoyButton>
    </template>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <ChoyCard :title="_t('Welcome')" class="md:col-span-2">
        <p class="text-sm text-muted-foreground">
          {{ _t('Welcome back. Use the sidebar to open applications, or jump to a shortcut below.') }}
        </p>
      </ChoyCard>

      <ChoyCard :title="_t('Status')">
        <dl class="grid gap-2 text-sm">
          <div class="flex justify-between gap-3">
            <dt class="text-muted-foreground">{{ _t('Theme') }}</dt>
            <dd class="font-medium">{{ themeLabel }}</dd>
          </div>
          <div class="flex justify-between gap-3">
            <dt class="text-muted-foreground">{{ _t('Density') }}</dt>
            <dd class="font-medium">{{ densityLabel }}</dd>
          </div>
        </dl>
      </ChoyCard>

      <ChoyCard :title="_t('Shortcuts')" class="md:col-span-2 xl:col-span-3">
        <ul v-if="shortcuts.length" class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <li v-for="item in shortcuts" :key="item.id">
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-md border border-border bg-background px-3 py-2 text-start text-sm hover:bg-muted"
              @click="openShortcut(item)"
            >
              <span class="font-medium">{{ item.label }}</span>
              <span v-if="item.path" class="ms-auto truncate text-xs text-muted-foreground">{{ item.path }}</span>
            </button>
          </li>
        </ul>
        <p v-else class="text-sm text-muted-foreground">
          {{ _t('No menu shortcuts yet. Install applications to populate the sidebar.') }}
        </p>
      </ChoyCard>
    </div>
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed, onActivated, ref } from 'vue';
import { useRouter } from 'vue-router';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyCard from '@/web/web/components/layout/ChoyCard.vue';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import {
  applyChoyThemePreference,
  readChoyThemePreference
} from '@/web/web/composables/applyChoyThemePreference';
import { useMenuStore } from '@/web/web/stores/menuStore';
import { createTranslate, translateTerm } from '@/web/web/i18n';
import { useI18n } from 'vue-i18n';
import type { MenuItem } from '@/core/web/menu';
import { collectHomeShortcuts, type HomeShortcut } from './homeShortcuts';

const { _t } = createTranslate('web', { scope: 'web/pages/HomeView' });

const prefs = ref(readChoyThemePreference());

const themeLabel = computed(() => prefs.value.theme ?? 'light');
const densityLabel = computed(() => prefs.value.density ?? 'comfortable');

let listMenus: () => MenuItem[] = () => [];
let labelOf: (item: MenuItem) => string = item => String(item.title || item.path || '');
let pushPath: (path: string) => void = () => {};

try {
  const menuStore = useMenuStore();
  listMenus = () => menuStore.getMenus();
} catch {
  // Unit mounts may omit pinia.
}

try {
  const composer = useI18n({ useScope: 'global' });
  labelOf = item => String(translateTerm(composer, item.titleText, item.title) || item.title || item.path || '');
} catch {
  // Unit mounts may omit vue-i18n.
}

try {
  const router = useRouter();
  if (router) {
    pushPath = path => {
      void router.push(path);
    };
  }
} catch {
  // Unit mounts may omit vue-router.
}

const shortcuts = computed(() => collectHomeShortcuts(listMenus(), 9, labelOf));

function openShortcut(item: HomeShortcut) {
  if (item.path) pushPath(item.path);
}

// Home is keepAlive; re-read persisted prefs when the cached view is shown again.
onActivated(() => {
  prefs.value = readChoyThemePreference();
});

function refresh() {
  applyChoyThemePreference(readChoyThemePreference());
  prefs.value = readChoyThemePreference();
}
</script>
