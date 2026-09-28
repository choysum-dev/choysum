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
      <div class="flex h-12 items-center gap-3 px-4 text-sm">
        <span class="font-semibold tracking-tight">Choysum</span>
        <div data-anchor="choy.shell.header-actions" class="ms-auto flex items-center gap-1">
          <slot name="header-actions" />
        </div>
      </div>
    </template>
    <template v-if="showSidebar && $slots.aside" #aside>
      <nav class="p-3 text-sm text-foreground/80" aria-label="Main">
        <slot name="aside" />
      </nav>
    </template>
    <slot>
      <router-view v-slot="{ Component, route }">
        <!-- KeepAlive stays mounted so cached views survive non-keepAlive navigations. -->
        <KeepAlive>
          <component
            :is="Component"
            v-if="Component && route.meta?.keepAlive"
            :key="route.path"
          />
        </KeepAlive>
        <component
          :is="Component"
          v-if="Component && !route.meta?.keepAlive"
          :key="route.fullPath"
        />
      </router-view>
    </slot>
    <template v-if="showFooter && $slots.footer" #footer>
      <div class="px-4 py-2 text-xs text-foreground/60">
        <slot name="footer" />
      </div>
    </template>
  </ChoyLayout>
</template>

<script setup lang="ts">
import { KeepAlive } from 'vue';
import ChoyLayout from './ChoyLayout.vue';

/**
 * Web product shell around ChoyLayout + router-view: brand bar,
 * header-actions anchor, and KeepAlive-aware router host.
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
