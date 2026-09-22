<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { Check } from 'lucide-vue-next';
import { SelectItem, SelectItemIndicator, SelectItemText } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{
  class?: ClassValue;
  value: string;
  disabled?: boolean;
}>();
</script>

<template>
  <SelectItem
    data-slot="select-item"
    :value="value"
    :disabled="disabled"
    :class="
      cn(
        'relative flex w-full cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none',
        'focus:bg-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        $props.class,
      )
    "
  >
    <span class="absolute left-2 flex size-3.5 items-center justify-center">
      <SelectItemIndicator>
        <Check class="size-4" />
      </SelectItemIndicator>
    </span>
    <SelectItemText>
      <slot />
    </SelectItemText>
  </SelectItem>
</template>
