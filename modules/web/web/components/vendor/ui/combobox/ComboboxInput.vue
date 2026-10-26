<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxInput } from 'reka-ui';
import { cn, controlFocusRing, type ClassValue } from '../../../../lib/utils';

withDefaults(
  defineProps<{
    class?: ClassValue;
    placeholder?: string;
    disabled?: boolean;
    /**
     * default — bordered form control; bare — panel filter (no border / focus ring).
     */
    variant?: 'default' | 'bare';
    /** Map the selected model value to the closed-state input text (e.g. field label). */
    displayValue?: (value: string) => string;
  }>(),
  {
    variant: 'default',
  },
);
</script>

<template>
  <ComboboxInput
    data-slot="combobox-input"
    :placeholder="placeholder"
    :disabled="disabled"
    :display-value="displayValue"
    :class="
      cn(
        variant === 'bare'
          ? 'h-control w-full min-w-0 flex-1 border-0 bg-transparent px-0 py-0 text-sm text-foreground shadow-none outline-none focus-visible:border-transparent focus-visible:ring-0'
          : [
              'flex h-control w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm text-foreground shadow-xs',
              'placeholder:text-muted-foreground',
              controlFocusRing,
            ],
        'disabled:cursor-not-allowed disabled:opacity-50',
        $props.class,
      )
    "
  />
</template>
