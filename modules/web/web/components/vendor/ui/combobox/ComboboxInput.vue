<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxInput } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

defineProps<{
  class?: ClassValue;
  placeholder?: string;
  /** Map the selected model value to the closed-state input text (e.g. field label). */
  displayValue?: (value: string) => string;
}>();
</script>

<template>
  <ComboboxInput
    data-slot="combobox-input"
    :placeholder="placeholder"
    :display-value="displayValue"
    :class="
      cn(
        'flex h-control w-full rounded-md border border-border bg-background px-3 py-1 text-sm text-foreground shadow-sm',
        'placeholder:text-foreground/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        $props.class,
      )
    "
  />
</template>
