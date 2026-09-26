<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, inject, type Ref } from 'vue';
import type { ChartConfig } from './chartTypes';

/**
 * Simple legend listing ChartConfig keys with color swatches.
 */
const configRef = inject<Ref<ChartConfig> | undefined>('choyChartConfig', undefined);

const items = computed(() => {
  const cfg = configRef?.value || {};
  return Object.entries(cfg).map(([key, item]) => ({
    key,
    label: item.label || key,
    color: item.color || `var(--color-${key})`,
  }));
});
</script>

<template>
  <div
    data-slot="chart-legend"
    class="flex flex-wrap items-center justify-center gap-3 pt-1"
  >
    <div
      v-for="item in items"
      :key="item.key"
      class="flex items-center gap-1.5"
    >
      <span
        class="inline-block size-2.5 shrink-0 rounded-[2px]"
        :style="{ backgroundColor: item.color }"
      />
      <span class="text-muted-foreground">{{ item.label }}</span>
    </div>
  </div>
</template>
