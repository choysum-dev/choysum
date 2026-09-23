<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import Button from '../vendor/ui/button/Button.vue';
import type { ClassValue } from '../../lib/utils';

type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'link';
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon';

/**
 * Public L1 button for custom pages (login, dogfood). Wraps L2 ui/button so
 * domain modules never import vendor/ui directly.
 */
const props = withDefaults(
  defineProps<{
    variant?: ButtonVariant;
    size?: ButtonSize;
    as?: string;
    class?: ClassValue;
    disabled?: boolean;
    type?: 'button' | 'submit' | 'reset';
  }>(),
  {
    variant: 'default',
    size: 'default',
    as: 'button',
    type: 'button',
  },
);
</script>

<template>
  <Button
    data-anchor="choy.button"
    :variant="props.variant"
    :size="props.size"
    :as="props.as"
    :class="props.class"
    :disabled="props.disabled"
    :type="props.type"
  >
    <slot />
  </Button>
</template>
