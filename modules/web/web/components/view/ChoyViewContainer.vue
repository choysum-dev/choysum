<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<!-- Region spacing (header ↔ body ↔ footer) is flex gap on the root; shell owns page gutters. -->
<template>
  <div
    data-slot="view-container"
    :class="
      cn(
        'flex h-full min-h-0 w-full min-w-0 flex-1 flex-col',
        !hasGapUtility(props.class) ? 'gap-3' : undefined,
        props.class,
      )
    "
  >
    <div v-if="$slots.header" data-slot="view-header" class="shrink-0">
      <slot name="header" />
    </div>
    <div data-slot="view-body" class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <slot />
    </div>
    <div v-if="$slots.footer" data-slot="view-footer" class="shrink-0">
      <slot name="footer" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { cn, type ClassValue } from '../../lib/utils';

defineOptions({ name: 'ChoyViewContainer' });

const props = defineProps<{
  class?: ClassValue;
}>();

/** True when a class token is a full (both-axis) Tailwind `gap-*` utility. */
function isFullGapToken(token: string): boolean {
  const base = token.includes(':') ? token.slice(token.lastIndexOf(':') + 1) : token;
  // Axis-specific gap-x-*/gap-y-* must not suppress the container's default gap-3.
  return /^(?:!)?gap-(?!x-|y-)/.test(base);
}

/** True when `class` already sets a Tailwind gap-* (so the default gap-3 is skipped). */
function hasGapUtility(value: ClassValue | undefined): boolean {
  if (value == null || value === false || value === '') return false;
  if (typeof value === 'string') return value.split(/\s+/).some((part) => isFullGapToken(part));
  if (typeof value === 'number') return false;
  if (Array.isArray(value)) return value.some((item) => hasGapUtility(item as ClassValue));
  if (typeof value === 'object') {
    return Object.entries(value).some(([key, on]) => !!on && isFullGapToken(key));
  }
  return false;
}
</script>
