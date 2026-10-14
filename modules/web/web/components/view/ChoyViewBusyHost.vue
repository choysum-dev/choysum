<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-slot="view-body-host"
    :class="hostClass"
    v-bind="hostAttrs"
  >
    <slot />
    <div
      v-if="busy"
      role="status"
      aria-live="polite"
      class="sr-only"
    >
      {{ busyLabel }}
    </div>
    <div
      v-if="busy"
      data-slot="view-busy-overlay"
      class="absolute inset-0 z-10 flex items-center justify-center bg-background/50"
      aria-hidden="true"
    >
      <ChoySpinner class="size-6 text-muted-foreground" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ChoySpinner from '../layout/ChoySpinner.vue';
import type { ClassValue } from '../../lib/utils';
import { viewBusyHostAttrs, viewBusyHostClass } from './viewBusy';

defineOptions({ name: 'ChoyViewBusyHost' });

const props = withDefaults(
  defineProps<{
    busy?: boolean;
    class?: ClassValue;
    /** Polite live-region text announced while busy (overlay stays aria-hidden). */
    busyLabel?: string;
  }>(),
  { busy: false, busyLabel: 'Loading' },
);

const busy = computed(() => !!props.busy);
const busyLabel = computed(() => props.busyLabel || 'Loading');
const hostClass = computed(() => viewBusyHostClass(busy.value, props.class));
const hostAttrs = computed(() => viewBusyHostAttrs(busy.value));
</script>
