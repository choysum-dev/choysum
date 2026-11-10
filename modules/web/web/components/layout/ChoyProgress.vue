<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ProgressRoot
    data-slot="progress"
    data-testid="choy-progress"
    :model-value="clamped"
    :class="cn('bg-primary/20 relative h-2 w-full overflow-hidden rounded-full', props.class)"
  >
    <ProgressIndicator
      data-slot="progress-indicator"
      class="bg-primary h-full w-full flex-1 transition-all"
      :style="`transform: translateX(-${100 - clamped}%);`"
    />
  </ProgressRoot>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ProgressIndicator, ProgressRoot } from 'reka-ui';
import { cn, type ClassValue } from '../../lib/utils';

/**
 * Product progress bar over reka-ui Progress (shadcn-vue Progress).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    modelValue?: number | null;
  }>(),
  {
    modelValue: 0,
  },
);

const clamped = computed(() => {
  const n = Number(props.modelValue ?? 0);
  if (!Number.isFinite(n)) return 0;
  return Math.min(100, Math.max(0, n));
});
</script>
