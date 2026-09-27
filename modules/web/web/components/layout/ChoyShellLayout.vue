<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyLayout
    :show-header="showHeader"
    :show-aside="showSidebar && !!$slots.aside"
    :show-footer="showFooter && !!$slots.footer"
    class="min-h-screen"
  >
    <template v-if="showHeader" #header>
      <div class="flex h-12 items-center justify-between gap-3 px-4 text-sm">
        <span class="font-semibold tracking-tight">Choysum</span>
        <slot name="header-actions" />
      </div>
    </template>
    <template v-if="showSidebar && $slots.aside" #aside>
      <nav class="p-3 text-sm text-foreground/80" aria-label="Main">
        <slot name="aside" />
      </nav>
    </template>
    <slot>
      <router-view />
    </slot>
    <template v-if="showFooter && $slots.footer" #footer>
      <div class="px-4 py-2 text-xs text-foreground/60">
        <slot name="footer" />
      </div>
    </template>
  </ChoyLayout>
</template>

<script setup lang="ts">
import ChoyLayout from './ChoyLayout.vue';

/**
 * Product shell around ChoyLayout + router-view. Replaces the Element Plus
 * OLayout host for web-owned routes during cutover.
 */
withDefaults(
  defineProps<{
    showHeader?: boolean;
    showSidebar?: boolean;
    showFooter?: boolean;
  }>(),
  {
    showHeader: true,
    showSidebar: false,
    showFooter: false,
  },
);
</script>
