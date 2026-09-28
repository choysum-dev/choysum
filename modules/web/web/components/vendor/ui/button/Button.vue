<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import { cn, type ClassValue } from '../../../../lib/utils';

type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'link';
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon';

defineOptions({ inheritAttrs: false });

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

// Parent @click is an emit listener on this component (not a fallthrough attr) when
// the root is another Vue component; re-emit from the native host click.
const emit = defineEmits(['click']);

function handleClick(event: Event) {
  if (props.disabled) {
    // Non-<button> hosts (e.g. <a href>) still navigate unless default is cancelled.
    event.preventDefault();
    return;
  }
  emit('click', event);
}

const attrs = useAttrs();

/** Non-button hosts: aria-disabled + out of tab order when disabled (disabled: CSS never matches). */
const nonButtonAttrs = computed(() =>
  props.disabled ? { ...attrs, 'aria-disabled': true, tabindex: -1 } : attrs,
);

const variantClass: Record<ButtonVariant, string> = {
  default: 'bg-primary text-background hover:opacity-90',
  secondary: 'bg-muted text-foreground hover:opacity-90',
  outline: 'border border-border bg-background hover:bg-muted',
  ghost: 'hover:bg-muted',
  destructive: 'bg-danger text-background hover:opacity-90',
  link: 'text-primary underline-offset-4 hover:underline',
};

const sizeClass: Record<ButtonSize, string> = {
  default: 'h-9 px-4 py-2',
  sm: 'h-8 rounded-md px-3 text-xs',
  lg: 'h-10 rounded-md px-6',
  icon: 'h-9 w-9',
};

const classes = computed(() =>
  cn(
    'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors',
    'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
    'disabled:pointer-events-none disabled:opacity-50',
    variantClass[props.variant],
    sizeClass[props.size],
    props.class,
  ),
);
</script>

<template>
  <button
    v-if="as === 'button'"
    v-bind="$attrs"
    data-slot="button"
    :class="classes"
    :disabled="disabled"
    :type="type"
    @click="handleClick"
  >
    <slot />
  </button>
  <component
    :is="as"
    v-else
    v-bind="nonButtonAttrs"
    data-slot="button"
    :class="classes"
    @click="handleClick"
  >
    <slot />
  </component>
</template>
