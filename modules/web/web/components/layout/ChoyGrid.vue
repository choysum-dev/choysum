<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.grid"
    :data-responsive="responsiveOn ? 'true' : 'false'"
    :class="cn('choy-grid grid w-full', gapClass, props.class)"
    :style="style"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed, provide } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import {
  ChoyGridColsKey,
  ChoyGridResponsiveKey,
  normalizeChoyGridCols,
} from './choyGridContext';

/**
 * CSS grid host for ChoyCol children.
 * Track count is the `cols` prop (not Tailwind `grid-cols-*` on `class`).
 * Default `responsive` remaps ChoyCol spans by this element's container width (421).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    cols?: number;
    gap?: 'none' | 'sm' | 'md' | 'lg';
    /** When true (default), ChoyCol spans follow container 421 bands. */
    responsive?: boolean;
  }>(),
  {
    cols: 12,
    gap: 'md',
    responsive: true,
  },
);

const safeCols = computed(() => normalizeChoyGridCols(props.cols));
const responsiveOn = computed(() => props.responsive !== false);

provide(ChoyGridColsKey, safeCols);
provide(ChoyGridResponsiveKey, responsiveOn);

const gapClass = computed(() => {
  switch (props.gap) {
    case 'none':
      return 'gap-0';
    case 'sm':
      return 'gap-2';
    case 'lg':
      return 'gap-6';
    default:
      // Tighter than gap-3 so form field blocks match --choy-field-group-gap.
      return 'gap-2';
  }
});

const style = computed(() => ({
  gridTemplateColumns: `repeat(${safeCols.value}, minmax(0, 1fr))`,
}));
</script>

<style scoped>
/* Named so ChoyCol span CQ does not match nested choy-col containers. */
.choy-grid {
  container-type: inline-size;
  container-name: choy-grid;
}
</style>
