<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyPage :title="_t('Home')" padding width="full">
    <template #title-actions>
      <ChoyButton type="button" variant="outline" @click="refresh">
        {{ _t('Refresh') }}
      </ChoyButton>
    </template>

    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      <ChoyCard :title="_t('Welcome')">
        <p class="text-sm text-foreground/80">
          {{ _t('Choysum web shell is running on the Choy UI kit.') }}
        </p>
      </ChoyCard>
      <ChoyCard :title="_t('Status')">
        <dl class="grid gap-2 text-sm">
          <div class="flex justify-between gap-3">
            <dt class="text-foreground/70">{{ _t('Theme') }}</dt>
            <dd>{{ themeLabel }}</dd>
          </div>
          <div class="flex justify-between gap-3">
            <dt class="text-foreground/70">{{ _t('Density') }}</dt>
            <dd>{{ densityLabel }}</dd>
          </div>
        </dl>
      </ChoyCard>
      <ChoyCard :title="_t('Next')">
        <p class="text-sm text-foreground/80">
          {{ _t('Domain modules still migrate onto Choy* in later cutover slices.') }}
        </p>
      </ChoyCard>
    </div>
  </ChoyPage>
</template>

<script setup lang="ts">
import { computed, onActivated, ref } from 'vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import ChoyCard from '@/web/web/components/layout/ChoyCard.vue';
import ChoyPage from '@/web/web/components/layout/ChoyPage.vue';
import {
  applyChoyThemePreference,
  readChoyThemePreference,
} from '@/web/web/composables/applyChoyThemePreference';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/pages/HomeView' });

const prefs = ref(readChoyThemePreference());

const themeLabel = computed(() => prefs.value.theme ?? 'light');
const densityLabel = computed(() => prefs.value.density ?? 'comfortable');

// Home is keepAlive; re-read persisted prefs when the cached view is shown again.
onActivated(() => {
  prefs.value = readChoyThemePreference();
});

function refresh() {
  applyChoyThemePreference(readChoyThemePreference());
  // Re-read so labels match whatever apply normalized into storage.
  prefs.value = readChoyThemePreference();
}
</script>
