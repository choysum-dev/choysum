<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Blocks the viewport while a navigation ticket is open and no surface has
    taken over — covers Guest→App shell swap so the destination does not look idle.
  -->
  <Teleport to="body">
    <div
      v-if="busy"
      data-testid="choy-navigation-handoff"
      data-anchor="choy.loading.navigation-handoff"
      class="choy-navigation-handoff fixed inset-0 z-[90] flex items-center justify-center bg-background/80"
      aria-hidden="true"
    >
      <ChoySpinner class="size-6 text-primary" />
    </div>
    <div v-if="busy" role="status" class="sr-only">{{ label }}</div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useNavigationHandoffBusy } from '@/core/web/client/loadingRegistry';
import { createTranslate } from '@/web/web/i18n';
import ChoySpinner from './ChoySpinner.vue';

defineOptions({ name: 'ChoyNavigationHandoff' });

const { _t } = createTranslate('web', { scope: 'web/components/layout/NavigationHandoff' });
const busy = useNavigationHandoffBusy();
const label = computed(() => _t('Loading...'));
</script>
