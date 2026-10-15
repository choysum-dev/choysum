<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div data-slot="view-container" :class="cn('flex h-full min-h-0 w-full min-w-0 flex-1 flex-col', props.class)">
    <div v-if="$slots.header" data-slot="view-header" class="shrink-0">
      <slot name="header" />
    </div>
    <div data-slot="view-body" class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <slot />
    </div>
    <div v-if="$slots.footer" data-slot="view-footer" class="shrink-0">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { cn, type ClassValue } from '../../lib/utils';

defineOptions({ name: 'ChoyViewContainer' });

const props = defineProps<{
  class?: ClassValue;
}>();
</script>
