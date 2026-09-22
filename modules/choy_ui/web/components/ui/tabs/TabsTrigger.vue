<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TabsTrigger } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{
  class?: ClassValue;
  value: string;
  disabled?: boolean;
}>();
</script>

<template>
  <TabsTrigger
    data-slot="tabs-trigger"
    :value="value"
    :disabled="disabled"
    :class="
      cn(
        'inline-flex items-center justify-center whitespace-nowrap rounded-sm px-3 py-1 text-sm font-medium',
        'ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        'disabled:pointer-events-none disabled:opacity-50',
        'data-[state=active]:bg-background data-[state=active]:text-foreground data-[state=active]:shadow-sm',
        $props.class,
      )
    "
  >
    <slot />
  </TabsTrigger>
</template>
