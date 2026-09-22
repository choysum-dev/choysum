<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next';
import { SelectIcon, SelectTrigger, SelectValue } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{ class?: ClassValue; placeholder?: string }>();
</script>

<template>
  <SelectTrigger
    data-slot="select-trigger"
    :class="
      cn(
        'flex h-9 w-full items-center justify-between rounded-md border border-border bg-background px-3 py-2 text-sm',
        'focus:outline-none focus:ring-2 focus:ring-ring disabled:cursor-not-allowed disabled:opacity-50',
        $props.class,
      )
    "
  >
    <SelectValue :placeholder="placeholder" />
    <SelectIcon as-child>
      <ChevronDown class="size-4 opacity-50" />
    </SelectIcon>
  </SelectTrigger>
</template>
