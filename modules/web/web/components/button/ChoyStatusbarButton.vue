<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyActionButton
    v-bind="$attrs"
    :variant="props.variant"
    :size="props.size"
    :as="props.as"
    :class="resolvedClass"
    :disabled="props.disabled"
    :busy="props.busy"
    :type="props.type"
    :action-id="props.actionId"
    :has-action="props.hasAction"
    :action-mode="props.actionMode"
    :requires-connectivity="props.requiresConnectivity"
    :confirm="props.confirm"
    :run="props.run"
    @execute="emit('execute')"
  >
    <slot />
  </ChoyActionButton>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ChoyActionButton from './ChoyActionButton.vue';
import { cn, type ClassValue } from '../../lib/utils';
import type { ChoyActionConfirm, ChoyActionMode } from '../../composables/useChoyActionPolicy';

/** Allowed visual variants for Form statusbar domain CTAs. */
export type ChoyStatusbarVariant =
  | 'outline-primary'
  | 'outline-destructive'
  | 'outline'
  | 'ghost';

/** Same size tokens as ChoyActionButton / ChoyButton. */
export type ChoyStatusbarButtonSize =
  | 'default'
  | 'sm'
  | 'lg'
  | 'icon'
  | 'xs'
  | 'icon-xs'
  | 'icon-sm'
  | 'icon-lg';

/**
 * Form statusbar domain CTA defaults: size sm + neutral outline (border-input).
 * Quiet at rest (Odoo-like peer actions); hover uses brand wash instead of accent gray.
 * Use variant=outline-destructive for deactivate / uninstall; outline-primary only when
 * a host truly needs a branded cage.
 */
defineOptions({ name: 'ChoyStatusbarButton', inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    variant?: ChoyStatusbarVariant;
    size?: ChoyStatusbarButtonSize;
    as?: string;
    class?: ClassValue;
    disabled?: boolean;
    busy?: boolean;
    type?: 'button' | 'submit' | 'reset';
    actionId?: string;
    hasAction?: (id: string | undefined) => boolean;
    actionMode?: ChoyActionMode;
    requiresConnectivity?: boolean;
    confirm?: ChoyActionConfirm;
    run?: () => void | Promise<void>;
  }>(),
  {
    variant: 'outline',
    size: 'sm',
    as: 'button',
    type: 'button',
    actionMode: 'hide',
    requiresConnectivity: true,
  },
);

const emit = defineEmits<{
  execute: [];
}>();

/**
 * Default outline: brand hover (not accent gray).
 * outline-primary: keep brand border, force foreground resting text.
 */
const resolvedClass = computed(() =>
  cn(
    props.variant === 'outline'
      ? 'hover:bg-primary/10 hover:text-primary'
      : props.variant === 'outline-primary'
        ? 'text-foreground hover:text-primary'
        : undefined,
    props.class,
  ),
);
</script>
