<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyButton
    v-if="aclVisible"
    v-bind="$attrs"
    :variant="props.variant"
    :size="props.size"
    :as="props.as"
    :class="props.class"
    :disabled="effectiveDisabled"
    :type="props.type"
    @click="onClick"
  >
    <slot />
  </ChoyButton>
</template>

<script setup lang="ts">
import ChoyButton from './ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import {
  useChoyActionPolicy,
  type ChoyActionConfirm,
  type ChoyActionMode,
} from '../../composables/useChoyActionPolicy';

type ButtonVariant =
  | 'default'
  | 'secondary'
  | 'outline'
  | 'outline-primary'
  | 'outline-destructive'
  | 'soft-primary'
  | 'soft-destructive'
  | 'ghost'
  | 'destructive'
  | 'link';
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon' | 'xs' | 'icon-xs' | 'icon-sm' | 'icon-lg';

export type { ChoyActionConfirm };

/**
 * L3 action shell: ACL, confirm, busy, offline gate, then run/@execute.
 * Inner surface is always ChoyButton (L1). Does not call model methods by name.
 */
defineOptions({ name: 'ChoyActionButton', inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    variant?: ButtonVariant;
    size?: ButtonSize;
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
    variant: 'default',
    size: 'default',
    as: 'button',
    type: 'button',
    actionMode: 'hide',
    requiresConnectivity: true,
  },
);

const emit = defineEmits<{
  execute: [];
}>();

const { pending, aclVisible, offlineBlocked, effectiveDisabled, activate } = useChoyActionPolicy(
  props,
  () => emit('execute'),
);

async function onClick(event: Event) {
  // Nested hosts (kanban cards, list rows) must not receive the same click.
  // Skip stopPropagation for as=span label chrome so the parent <label> still opens the file picker.
  await activate(event, {
    stopPropagation: props.as === 'button' || effectiveDisabled.value,
  });
}

defineExpose({
  onClick,
  effectiveDisabled,
  aclVisible,
  offlineBlocked,
  pending,
});
</script>
