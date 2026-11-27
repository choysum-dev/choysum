<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-anchor="choy.col"
    :data-label-gutter="labelGutterOn ? 'true' : 'false'"
    :data-label-width="labelWidthOn ? 'true' : undefined"
    :class="cn('choy-col min-w-0', props.class)"
    :style="style"
  >
    <!--
      When label-gutter is on, the outer Col is the CQ size box (container-name
      choy-col); the inner host owns the 2-track stack so narrow CQ can collapse
      tracks (a container cannot query-style itself).
    -->
    <div v-if="labelGutterOn" class="choy-col__stack">
      <slot />
    </div>
    <slot v-else />
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
 * Grid column: 421 span bands plus optional shared label gutter for stacked fields.
 * `label-width` fixes the Col label track (not a per-field API).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    span?: ChoyColSpanInput;
    /** When true (default), fields in this Col share one label track. */
    labelGutter?: boolean;
    /** Optional fixed label track width (e.g. `8rem`); ignored when narrow → vertical. */
    labelWidth?: string;
  }>(),
  {
    span: undefined,
    labelGutter: true,
  },
);

const gridCols = inject(ChoyGridColsKey, undefined);
const gridResponsive = inject(ChoyGridResponsiveKey, undefined);

const labelGutterOn = computed(() => props.labelGutter !== false);
const labelWidthTrimmed = computed(() => String(props.labelWidth ?? '').trim());
const labelWidthOn = computed(() => labelGutterOn.value && labelWidthTrimmed.value !== '');

const style = computed(() => {
  const bands = resolveChoyColSpanBands(
    props.span,
    gridCols?.value,
    gridResponsive?.value !== false,
  );
  const out: Record<string, string> = {
    '--choy-col-span-base': String(bands.base),
    '--choy-col-span-md': String(bands.md),
    '--choy-col-span-lg': String(bands.lg),
  };
  if (labelWidthOn.value) {
    out['--choy-field-label-width'] = labelWidthTrimmed.value;
  }
  return out;
});
</script>

<style scoped>
/*
 * Span CQ targets named grid container (not this Col).
 * Breakpoints sync with --choy-grid-break-md/lg (36rem / 56rem); literals only.
 */
.choy-col {
  grid-column: span var(--choy-col-span-base) / span var(--choy-col-span-base);
}

@container choy-grid (min-width: 36rem) {
  .choy-col {
    grid-column: span var(--choy-col-span-md) / span var(--choy-col-span-md);
  }
}

@container choy-grid (min-width: 56rem) {
  .choy-col {
    grid-column: span var(--choy-col-span-lg) / span var(--choy-col-span-lg);
  }
}

/* Outer Col is the label CQ size box when gutter is on. Stack tracks live in index.css. */
.choy-col[data-label-gutter='true'] {
  container-type: inline-size;
  container-name: choy-col;
}

.choy-col__stack {
  min-width: 0;
}
</style>
