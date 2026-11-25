<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <DropdownMenuItem
    v-if="aclVisible"
    v-bind="$attrs"
    :class="props.class"
    :disabled="effectiveDisabled"
    :inset="props.inset"
    @select="onSelect"
  >
    <slot />
  </DropdownMenuItem>
</template>

<script setup lang="ts">
import type { ClassValue } from '../../lib/utils';
import DropdownMenuItem from '../vendor/ui/dropdown-menu/DropdownMenuItem.vue';
import {
  useChoyActionPolicy,
  type ChoyActionConfirm,
  type ChoyActionMode,
} from '../../composables/useChoyActionPolicy';

export type { ChoyActionConfirm };

/**
 * L3 action shell with DropdownMenuItem skin (same policy as ChoyActionButton).
 * Use inside DropdownMenuContent / page-io overflow slots — not as a toolbar button.
 */
defineOptions({ name: 'ChoyActionMenuItem', inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    inset?: boolean;
    disabled?: boolean;
    busy?: boolean;
    actionId?: string;
    hasAction?: (id: string | undefined) => boolean;
    actionMode?: ChoyActionMode;
    requiresConnectivity?: boolean;
    confirm?: ChoyActionConfirm;
    run?: () => void | Promise<void>;
  }>(),
  {
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

async function onSelect(event: Event) {
  await activate(event);
}

defineExpose({
  onSelect,
  effectiveDisabled,
  aclVisible,
  offlineBlocked,
  pending,
});
</script>
