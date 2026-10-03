<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import { cn, type ClassValue } from '../../../../lib/utils';

type ButtonVariant = 'default' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'link';
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon' | 'xs' | 'icon-xs' | 'icon-sm' | 'icon-lg';

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
    // Non-<button> hosts still navigate unless default is cancelled, and must not
    // bubble to ancestor handlers the way a native disabled button would not.
    // `as` may be a component whose `click` emit passes a non-DOM payload.
    const e = event as Partial<Event> | undefined;
    e?.preventDefault?.();
    e?.stopPropagation?.();
    return;
  }
  emit('click', event);
}

const attrs = useAttrs();

/** Non-button hosts: drop navigation targets when disabled; add noopener for _blank. */
const nonButtonAttrs = computed(() => {
  const rest: Record<string, unknown> = { ...attrs };
  if (props.disabled) {
    delete rest.href;
    delete rest.to;
    delete rest.target;
    delete rest.rel;
    return { ...rest, 'aria-disabled': true, tabindex: -1 };
  }
  if (rest.target === '_blank') {
    const currentRel = String(rest.rel ?? '').trim();
    if (!/\bnoopener\b/i.test(currentRel)) {
      rest.rel = currentRel ? `${currentRel} noopener` : 'noopener noreferrer';
    }
  }
  return rest;
});

const variantClass: Record<ButtonVariant, string> = {
  default: 'bg-primary text-primary-foreground hover:bg-primary/90',
  secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
  outline: 'border border-border bg-background hover:bg-accent hover:text-accent-foreground',
  ghost: 'hover:bg-accent hover:text-accent-foreground',
  destructive: 'bg-destructive text-primary-foreground hover:bg-destructive/90',
  link: 'text-primary underline-offset-4 hover:underline',
};

const sizeClass: Record<ButtonSize, string> = {
  default: 'h-control px-4 py-2',
  sm: 'h-control-sm rounded-md px-3 text-xs',
  lg: 'h-control-lg rounded-md px-6',
  icon: 'size-control',
  xs: 'h-control-sm gap-1 rounded-md px-2 text-xs',
  'icon-xs': 'size-control rounded-md',
  'icon-sm': 'size-control',
  'icon-lg': 'size-control',
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
    :class="[classes, disabled ? 'pointer-events-none opacity-50' : '']"
    @click="handleClick"
  >
    <slot />
  </component>
</template>
