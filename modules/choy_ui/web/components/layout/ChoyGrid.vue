<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, provide } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import { ChoyGridColsKey } from './choyGridContext';

/**
 * CSS grid host for ChoyCol children.
 * Track count is the `cols` prop (not Tailwind `grid-cols-*` on `class`);
 * change `cols` / `span` for layout — inline styles intentionally own the tracks.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    cols?: number;
    gap?: 'none' | 'sm' | 'md' | 'lg';
  }>(),
  {
    cols: 12,
    gap: 'md',
  },
);

const safeCols = computed(() => {
  const n = Number(props.cols);
  if (!Number.isFinite(n)) {
    return 12;
  }
  return Math.max(1, Math.floor(n));
});

provide(ChoyGridColsKey, safeCols);

const gapClass = computed(() => {
  switch (props.gap) {
    case 'none':
      return 'gap-0';
    case 'sm':
      return 'gap-2';
    case 'lg':
      return 'gap-6';
    default:
      return 'gap-4';
  }
});

const style = computed(() => ({
  gridTemplateColumns: `repeat(${safeCols.value}, minmax(0, 1fr))`,
}));
</script>

<template>
  <div
    data-anchor="choy.grid"
    :class="cn('choy-grid grid w-full', gapClass, props.class)"
    :style="style"
  >
    <slot />
  </div>
</template>
