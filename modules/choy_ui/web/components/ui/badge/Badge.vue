<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import { cn, type ClassValue } from '../../../lib/utils';

type BadgeVariant = 'default' | 'secondary' | 'outline' | 'destructive';

const props = withDefaults(
  defineProps<{
    variant?: BadgeVariant;
    class?: ClassValue;
  }>(),
  { variant: 'default' },
);

const variantClass: Record<BadgeVariant, string> = {
  default: 'border-transparent bg-primary text-background',
  secondary: 'border-transparent bg-muted text-foreground',
  outline: 'border-border text-foreground',
  destructive: 'border-transparent bg-danger text-background',
};

const classes = computed(() =>
  cn(
    'inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors',
    variantClass[props.variant],
    props.class,
  ),
);
</script>

<template>
  <span data-slot="badge" :class="classes">
    <slot />
  </span>
</template>
