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
      :class="
        cn(
          'choy-layout__header shrink-0 border-b border-border bg-background',
          asideOverlay ? 'relative z-50' : '',
        )
      "
    >
      <slot name="header" />
    </header>
    <div class="choy-layout__body relative flex min-h-0 flex-1">
      <div
        v-if="asideOverlay && (showAside ?? !!$slots.aside)"
        data-anchor="choy.layout.aside-backdrop"
        data-testid="choy-layout-aside-backdrop"
        class="fixed inset-0 z-40 bg-foreground/40"
        aria-hidden="true"
        @click="emit('aside-dismiss')"
      />
      <aside
        v-if="showAside ?? !!$slots.aside"
        ref="asideEl"
        data-anchor="choy.layout.aside"
        data-testid="choy-layout-aside"
        :role="asideOverlay ? 'dialog' : undefined"
        :aria-modal="asideOverlay ? 'true' : undefined"
        :aria-label="asideOverlay ? asideAriaLabel : undefined"
        :tabindex="asideOverlay ? -1 : undefined"
        :class="
          cn(
            'choy-layout__aside flex shrink-0 flex-col border-border bg-background',
            asideOverlay
              ? 'fixed inset-y-0 start-0 z-50 border-e shadow-lg outline-none'
              : 'relative border-e',
          )
        "
        :style="{ width: asideWidth }"
      >
        <div class="choy-layout__aside-scroll min-h-0 flex-1 overflow-auto">
          <slot name="aside" />
        </div>
        <div
          v-if="$slots['aside-foot']"
          data-anchor="choy.layout.aside-foot"
          class="choy-layout__aside-foot shrink-0 border-t border-border"
        >
          <slot name="aside-foot" />
        </div>
      </aside>
      <main
        data-anchor="choy.layout.main"
        class="choy-layout__main min-w-0 flex-1 overflow-auto"
        :inert="asideOverlay || undefined"
      >
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
import { computed, nextTick, ref, watch } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import { choyLayoutAsideWidth } from './choyLayoutAsideWidth';

/**
 * Application chrome: Top bar (header) + Nav rail (aside + optional aside-foot)
 * + Canvas (main). Attribution belongs in aside-foot, not a Canvas-wide footer.
 * Overlay mode exposes a modal dialog rail with backdrop dismiss.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    showHeader?: boolean;
    showAside?: boolean;
    showFooter?: boolean;
    /** When true, aside is a start-side overlay drawer (mobile). */
    asideOverlay?: boolean;
    /** expanded | collapsed width tokens; ignored when aside hidden. */
    asideCollapsed?: boolean;
    /** Accessible name when asideOverlay (dialog). */
    asideAriaLabel?: string;
  }>(),
  {
    showHeader: undefined,
    showAside: undefined,
    showFooter: undefined,
    asideOverlay: false,
    asideCollapsed: false,
    asideAriaLabel: 'Main navigation',
  },
);

const emit = defineEmits<{
  'aside-dismiss': [];
}>();

const asideEl = ref<HTMLElement | null>(null);

const asideWidth = computed(() => choyLayoutAsideWidth(!!props.asideCollapsed));

watch(
  () => props.asideOverlay && (props.showAside ?? true),
  async (open) => {
    if (!open) return;
    await nextTick();
    asideEl.value?.focus?.();
  },
);

defineExpose({ asideEl });
</script>
