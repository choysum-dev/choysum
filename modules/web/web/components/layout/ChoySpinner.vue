<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Color on the wrapper: global svg/.lucide use color:inherit !important. -->
  <span
    data-testid="choy-spinner"
    role="status"
    :aria-label="label || 'Loading'"
    :class="cn('choy-spinner inline-flex size-4 text-primary', props.class)"
  >
    <Loader2 class="size-full animate-spin" aria-hidden="true" />
  </span>
</template>

<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next';
import { cn, type ClassValue } from '../../lib/utils';

/**
 * Product loading spinner (Lucide Loader2).
 * Defaults to theme primary; pass text-* to override (e.g. text-muted-foreground).
 * Color is set on the wrapper so global `svg, .lucide { color: inherit !important }` can inherit it.
 */
const props = defineProps<{
  class?: ClassValue;
  label?: string;
}>();
</script>
