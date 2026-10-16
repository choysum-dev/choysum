<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { Toggle } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';
import { toggleVariants } from './toggleVariants';

const pressed = defineModel<boolean>('pressed', { default: false });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    variant?: 'default' | 'outline';
    size?: 'default' | 'sm' | 'lg';
    disabled?: boolean;
  }>(),
  {
    variant: 'default',
    size: 'default',
    disabled: false,
  },
);
</script>

<template>
  <Toggle
    v-slot="slotProps"
    v-model:pressed="pressed"
    data-slot="toggle"
    :disabled="disabled"
    :class="cn(toggleVariants({ variant, size }), props.class)"
  >
    <slot v-bind="slotProps" />
  </Toggle>
</template>
