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
import {
  ChoyGridColsKey,
  ChoyGridResponsiveKey,
  resolveChoyColSpanBands,
  type ChoyColSpanInput,
} from './choyGridContext';

/**
 * Grid column. Under a responsive ChoyGrid, omitted span is one 421 field slot
 * (3/6/12). A number is the wide (lg) target and scales up; an object sets
 * container bands explicitly (keys are container bands, not viewport).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    span?: ChoyColSpanInput;
  }>(),
  {
    span: undefined,
  },
);

const gridCols = inject(ChoyGridColsKey, undefined);
const gridResponsive = inject(ChoyGridResponsiveKey, undefined);

const style = computed(() => {
  const bands = resolveChoyColSpanBands(
    props.span,
    gridCols?.value,
    gridResponsive?.value !== false,
  );
  return {
    '--choy-col-span-base': String(bands.base),
    '--choy-col-span-md': String(bands.md),
    '--choy-col-span-lg': String(bands.lg),
  } as Record<string, string>;
});
</script>

<style scoped>
/*
 * Breakpoints must stay in sync with --choy-grid-break-md/lg in tokens.css
 * (36rem / 56rem). Literal lengths: @container min-width var() is unreliable.
 */
.choy-col {
  grid-column: span var(--choy-col-span-base) / span var(--choy-col-span-base);
}

@container (min-width: 36rem) {
  .choy-col {
    grid-column: span var(--choy-col-span-md) / span var(--choy-col-span-md);
  }
}

@container (min-width: 56rem) {
  .choy-col {
    grid-column: span var(--choy-col-span-lg) / span var(--choy-col-span-lg);
  }
}
</style>
