<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Teleport to="body">
    <div
      v-if="busy"
      class="module-op-progress-lock fixed inset-0 z-50 flex items-center justify-center bg-background/70 backdrop-blur-[1px]"
      data-testid="module-op-progress-strip"
      role="alertdialog"
      aria-modal="true"
      aria-busy="true"
      :aria-labelledby="title ? 'module-op-progress-title' : undefined"
      aria-describedby="module-op-progress-headline"
      @keydown.esc.prevent
      @pointerdown.prevent
    >
      <div
        class="flex max-w-[min(92vw,20rem)] flex-col items-center gap-3 px-4 text-center"
        data-testid="module-op-progress-panel"
      >
        <ChoySpinner class="size-8" :label="headline || title" />
        <div
          v-if="title"
          id="module-op-progress-title"
          class="text-sm font-medium text-foreground"
        >
          {{ title }}
        </div>
        <div
          id="module-op-progress-headline"
          class="text-sm text-muted-foreground"
          data-testid="module-op-progress-headline"
        >
          {{ headline }}
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ChoySpinner } from '@/web';

defineOptions({ name: 'ModuleCatalogOpProgressStrip' });

withDefaults(
  defineProps<{
    busy?: boolean;
    title?: string;
    headline?: string;
    /** Kept for host compatibility; Spinner path ignores numeric progress. */
    value?: number;
  }>(),
  {
    busy: false,
    title: '',
    headline: '',
    value: 0,
  },
);
</script>
