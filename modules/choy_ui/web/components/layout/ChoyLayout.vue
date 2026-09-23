<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, useSlots } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';

/**
 * Application shell (header / aside / main / footer). Product Header/Sidebar
 * wiring lands at cutover; isolation uses slots only.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    showHeader?: boolean;
    showAside?: boolean;
    showFooter?: boolean;
  }>(),
  {
    showHeader: undefined,
    showAside: undefined,
    showFooter: undefined,
  },
);

const slots = useSlots();

const headerVisible = computed(() => props.showHeader ?? !!slots.header);
const asideVisible = computed(() => props.showAside ?? !!slots.aside);
const footerVisible = computed(() => props.showFooter ?? !!slots.footer);
</script>

<template>
  <div
    data-anchor="choy.layout"
    :class="cn('choy-layout flex min-h-0 flex-1 flex-col bg-background text-foreground', props.class)"
  >
    <header v-if="headerVisible" data-anchor="choy.layout.header" class="choy-layout__header shrink-0 border-b border-border">
      <slot name="header" />
    </header>
    <div class="choy-layout__body flex min-h-0 flex-1">
      <aside
        v-if="asideVisible"
        data-anchor="choy.layout.aside"
        class="choy-layout__aside w-56 shrink-0 border-r border-border"
      >
        <slot name="aside" />
      </aside>
      <main data-anchor="choy.layout.main" class="choy-layout__main min-w-0 flex-1 overflow-auto">
        <slot />
      </main>
    </div>
    <footer v-if="footerVisible" data-anchor="choy.layout.footer" class="choy-layout__footer shrink-0 border-t border-border">
      <slot name="footer" />
    </footer>
  </div>
</template>
