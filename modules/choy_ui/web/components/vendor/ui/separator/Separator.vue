<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import { cn, type ClassValue } from '../../../../lib/utils';

const props = withDefaults(
  defineProps<{
    orientation?: 'horizontal' | 'vertical';
    decorative?: boolean;
    class?: ClassValue;
  }>(),
  {
    orientation: 'horizontal',
    decorative: true,
  },
);

const classes = computed(() =>
  cn(
    'shrink-0 bg-border',
    props.orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
    props.class,
  ),
);
</script>

<template>
  <div
    data-slot="separator"
    :role="decorative ? 'none' : 'separator'"
    :aria-orientation="decorative ? undefined : orientation"
    :aria-hidden="decorative ? true : undefined"
    :class="classes"
  />
</template>
