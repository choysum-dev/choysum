<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    role="group"
    data-slot="button-group"
    :class="
      cn(
        'inline-flex w-fit items-stretch',
        // Drop per-button shadow so adjacent outline borders share one edge.
        // Also join nested buttons (e.g. TooltipRoot wrapping each control).
        '[&>*]:shadow-none [&>*_button]:shadow-none',
        '[&>*:not(:first-child)]:rounded-s-none [&>*:not(:first-child)]:border-s-0',
        '[&>*:not(:first-child)_button]:rounded-s-none [&>*:not(:first-child)_button]:border-s-0',
        '[&>*:not(:last-child)]:rounded-e-none',
        '[&>*:not(:last-child)_button]:rounded-e-none',
        props.class,
      )
    "
    :aria-label="ariaLabel"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
import { cn, type ClassValue } from '../../lib/utils';

defineOptions({ name: 'ChoyButtonGroup' });

const props = defineProps<{
  class?: ClassValue;
  ariaLabel?: string;
}>();
</script>
