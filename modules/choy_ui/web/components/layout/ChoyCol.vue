<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, inject } from 'vue';
import { cn, type ClassValue } from '../../lib/utils';
import { ChoyGridColsKey } from './choyGridContext';

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
  const colsRaw = Number(gridCols?.value ?? 12);
  const cols = Number.isFinite(colsRaw) ? Math.max(1, Math.floor(colsRaw)) : 12;
  const spanRaw = props.span === undefined ? cols : Number(props.span);
  const spanBase = Number.isFinite(spanRaw) ? Math.floor(spanRaw) : cols;
  const span = Math.min(Math.max(spanBase, 1), cols);
  return { gridColumn: `span ${span} / span ${span}` };
});
</script>

<template>
  <div data-anchor="choy.col" :class="cn('choy-col min-w-0', props.class)" :style="style">
    <slot />
  </div>
</template>
