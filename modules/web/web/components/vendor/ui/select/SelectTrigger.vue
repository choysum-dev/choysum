<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next';
import { SelectIcon, SelectTrigger, SelectValue } from 'reka-ui';
import { cn, controlFocusRing, type ClassValue } from '../../../../lib/utils';

defineProps<{
  class?: ClassValue;
  placeholder?: string;
  id?: string;
  disabled?: boolean;
  'aria-invalid'?: boolean | 'true' | 'false';
  'aria-required'?: boolean | 'true' | 'false';
  'aria-describedby'?: string;
}>();
</script>

<template>
  <SelectTrigger
    data-slot="select-trigger"
    :id="id || undefined"
    :disabled="disabled"
    :aria-invalid="$props['aria-invalid']"
    :aria-required="$props['aria-required']"
    :aria-describedby="$props['aria-describedby']"
    :class="
      cn(
        'flex h-control w-full items-center justify-between rounded-md border border-input bg-transparent px-3 py-2 text-sm text-foreground shadow-xs',
        controlFocusRing,
        'disabled:cursor-not-allowed disabled:opacity-50',
        $props.class,
      )
    "
  >
    <!-- Empty default slot still occupies SelectValue and hides selected text. -->
    <SelectValue v-if="!$slots.default" :placeholder="placeholder" />
    <SelectValue v-else :placeholder="placeholder">
      <slot />
    </SelectValue>
    <SelectIcon as-child>
      <ChevronDown class="size-4 opacity-50" />
    </SelectIcon>
  </SelectTrigger>
</template>
