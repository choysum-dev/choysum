<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, provide, toRefs } from 'vue';
import { cn, type ClassValue } from '../../../../lib/utils';
import type { ChartConfig } from './chartTypes';

/**
 * Chart shell: injects --color-<key> CSS vars from ChartConfig for Unovis fills.
 */
const props = defineProps<{
  config: ChartConfig;
  class?: ClassValue;
}>();

const { config } = toRefs(props);
provide('choyChartConfig', config);

const styleVars = computed(() => {
  const style: Record<string, string> = {};
  for (const [key, item] of Object.entries(props.config || {})) {
    if (item?.color) {
      style[`--color-${key}`] = item.color;
    }
  }
  return style;
});
</script>

<template>
  <div
    data-slot="chart"
    :class="cn('flex w-full flex-col gap-2 text-xs text-foreground', $props.class)"
    :style="styleVars"
  >
    <slot />
  </div>
</template>
