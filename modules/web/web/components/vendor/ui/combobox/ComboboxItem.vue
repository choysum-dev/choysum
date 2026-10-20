<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { Check } from 'lucide-vue-next';
import { ComboboxItem, ComboboxItemIndicator } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{
  class?: ClassValue;
  value: string;
  disabled?: boolean;
}>();
</script>

<template>
  <ComboboxItem
    data-slot="combobox-item"
    :value="value"
    :disabled="disabled"
    :class="
      cn(
        'relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pe-8 ps-2 text-sm outline-none',
        'data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground',
        'data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        $props.class,
      )
    "
  >
    <span class="min-w-0 flex-1 truncate">
      <slot />
    </span>
    <span
      class="pointer-events-none absolute end-2 flex size-3.5 items-center justify-center"
    >
      <ComboboxItemIndicator>
        <Check class="size-4" />
      </ComboboxItemIndicator>
    </span>
  </ComboboxItem>
</template>
