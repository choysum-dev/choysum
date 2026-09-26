<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div data-anchor="choy.col" :class="cn('choy-col min-w-0', props.class)" :style="style">
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import { ChoyGridColsKey, resolveChoyColSpan } from './choyGridContext';

/**
 * Grid column. `span` defaults to full width of the parent ChoyGrid.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    span?: number;
  }>(),
  {
    span: undefined,
  },
);

const gridCols = inject(ChoyGridColsKey, undefined);

const style = computed(() => {
  const span = resolveChoyColSpan(props.span, gridCols?.value);
  return { gridColumn: `span ${span} / span ${span}` };
});
</script>
