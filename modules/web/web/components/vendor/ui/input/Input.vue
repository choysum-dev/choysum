<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import { cn, type ClassValue } from '../../../../lib/utils';

const props = defineProps<{
  class?: ClassValue;
  modelValue?: string;
}>();

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const classes = computed(() =>
  cn(
    'flex h-control w-full min-w-0 rounded-md border border-input bg-transparent px-3 py-1 text-sm text-foreground shadow-xs',
    'placeholder:text-muted-foreground outline-none transition-[color,box-shadow]',
    'focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50',
    'aria-invalid:border-destructive aria-invalid:ring-destructive/20',
    'disabled:pointer-events-none disabled:cursor-not-allowed disabled:opacity-50',
    props.class,
  ),
);

function onInput(event: Event): void {
  const target = event.target as HTMLInputElement;
  emit('update:modelValue', target.value);
}
</script>

<template>
  <input
    data-slot="input"
    :class="classes"
    :value="modelValue"
    v-bind="$attrs"
    @input="onInput"
  />
</template>
