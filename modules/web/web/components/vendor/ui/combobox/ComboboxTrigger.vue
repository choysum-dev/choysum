<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ComboboxTrigger } from 'reka-ui';
import { cn, controlFocusRing, type ClassValue } from '../../../../lib/utils';

defineProps<{
  class?: ClassValue;
  disabled?: boolean;
  id?: string;
  name?: string;
  'aria-invalid'?: boolean | 'true' | 'false';
  'aria-required'?: boolean | 'true' | 'false';
  'aria-describedby'?: string;
}>();
</script>

<template>
  <ComboboxTrigger
    data-slot="combobox-trigger"
    :id="id || undefined"
    :name="name || undefined"
    :disabled="disabled"
    :aria-invalid="$props['aria-invalid']"
    :aria-required="$props['aria-required']"
    :aria-describedby="$props['aria-describedby']"
    :class="
      cn(
        'flex h-control w-full items-center justify-between gap-2 rounded-md border border-input bg-transparent px-3 py-1 text-sm font-normal shadow-xs',
        'placeholder:text-muted-foreground',
        controlFocusRing,
        'disabled:cursor-not-allowed disabled:opacity-50',
        'aria-invalid:border-destructive aria-invalid:ring-destructive/20',
        $props.class,
      )
    "
  >
    <slot />
  </ComboboxTrigger>
</template>

<!-- Keep the same thin border-ring as focused Input / M2O while the portaled list holds focus. -->
<style>
  [data-slot='combobox-trigger'][data-state='open'] {
    border-color: var(--color-ring);
  }
</style>
