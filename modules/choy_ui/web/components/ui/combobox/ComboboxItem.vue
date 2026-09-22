<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxItem } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

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
        'relative flex cursor-default select-none items-center rounded-sm px-2 py-1.5 text-sm outline-none',
        'data-[highlighted]:bg-muted data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        $props.class,
      )
    "
  >
    <slot />
  </ComboboxItem>
</template>
