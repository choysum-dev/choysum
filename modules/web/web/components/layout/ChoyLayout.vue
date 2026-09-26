<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.layout"
    :class="cn('choy-layout flex min-h-0 flex-1 flex-col bg-background text-foreground', props.class)"
  >
    <header
      v-if="showHeader ?? !!$slots.header"
      data-anchor="choy.layout.header"
      class="choy-layout__header shrink-0 border-b border-border"
    >
      <slot name="header" />
    </header>
    <div class="choy-layout__body flex min-h-0 flex-1">
      <aside
        v-if="showAside ?? !!$slots.aside"
        data-anchor="choy.layout.aside"
        class="choy-layout__aside w-56 shrink-0 border-r border-border"
      >
        <slot name="aside" />
      </aside>
      <main data-anchor="choy.layout.main" class="choy-layout__main min-w-0 flex-1 overflow-auto">
        <slot />
      </main>
    </div>
    <footer
      v-if="showFooter ?? !!$slots.footer"
      data-anchor="choy.layout.footer"
      class="choy-layout__footer shrink-0 border-t border-border"
    >
      <slot name="footer" />
    </footer>
  </div>
</template>

<script setup lang="ts">
import { cn, type ClassValue } from '../../lib/utils';

/**
 * Application shell (header / aside / main / footer). Product Header/Sidebar
 * wiring lands at cutover; isolation uses slots only.
 * Slot visibility is read from `$slots` at render time (slots are not reactive).
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
</script>
