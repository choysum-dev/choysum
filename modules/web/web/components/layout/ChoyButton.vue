<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Button
    v-bind="{ ...$attrs, 'data-anchor': 'choy.button' }"
    :variant="props.variant"
    :size="props.size"
    :as="props.as"
    :class="props.class"
    :disabled="props.disabled"
    :type="props.type"
    @click="handleClick"
  >
    <slot />
  </Button>
</template>

<script setup lang="ts">
import Button from '../vendor/ui/button/Button.vue';
import type { ClassValue } from '../../lib/utils';

type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'link';
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon';

/**
 * Public L1 button for custom pages (login, dogfood). Wraps L2 ui/button so
 * domain modules never import vendor/ui directly.
 *
 * Declares click so parent @click is an emit listener (nested Button is a
 * component root; attrs onClick would not reach the native element).
 */
defineOptions({ name: 'ChoyButton', inheritAttrs: false });

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

const emit = defineEmits(['click']);

function handleClick(event: Event) {
  if (props.disabled) {
    // Nested Button / component hosts may emit a non-DOM click payload.
    const e = event as Partial<Event> | undefined;
    e?.preventDefault?.();
    e?.stopPropagation?.();
    return;
  }
  emit('click', event);
}
</script>
