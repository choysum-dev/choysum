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
import { computed, ref } from 'vue';
import ChoyButton from './ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import { confirmChoyAction } from '../../composables/confirmChoyAction';
import { getGlobalActionChecker } from '../../directives/action';
import { isConnectivityUnreachable } from '../../stores/connectivityStore';

type ButtonVariant =
  | 'default'
  | 'secondary'
  | 'outline'
  | 'outline-destructive'
  | 'ghost'
  | 'destructive'
  | 'link';
type ButtonSize = 'default' | 'sm' | 'lg' | 'icon' | 'xs' | 'icon-xs' | 'icon-sm' | 'icon-lg';
type ActionMode = 'hide' | 'disable';

export type ChoyActionConfirm =
  | string
  | {
      message: string;
      title?: string;
      destructive?: boolean;
      confirmText?: string;
      cancelText?: string;
    };

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
    actionMode?: ActionMode;
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

const pending = ref(false);

const actionIdTrimmed = computed(() => String(props.actionId ?? '').trim());

function isActionAllowedNow(): boolean {
  const id = actionIdTrimmed.value;
  if (!id) return true;
  const checker = props.hasAction ?? getGlobalActionChecker();
  if (typeof checker !== 'function') return true;
  return !!checker(id);
}

const aclAllowed = computed(() => isActionAllowedNow());

/** Hide mode + denied → do not mount; disable mode keeps the control. */
const aclVisible = computed(() => {
  if (!actionIdTrimmed.value) return true;
  if (aclAllowed.value) return true;
  return props.actionMode === 'disable';
});

const offlineBlocked = computed(
  () => props.requiresConnectivity !== false && isConnectivityUnreachable(),
);

const effectiveDisabled = computed(
  () =>
    !!props.disabled ||
    !!props.busy ||
    pending.value ||
    offlineBlocked.value ||
    (!!actionIdTrimmed.value && !aclAllowed.value && props.actionMode === 'disable'),
);

async function onClick(event: Event) {
  // Nested hosts (kanban cards, list rows) must not receive the same click.
  // Skip stopPropagation for as=span label chrome so the parent <label> still opens the file picker.
  const e = event as Partial<Event> | undefined;
  if (props.as === 'button' || effectiveDisabled.value) {
    e?.stopPropagation?.();
  }
  if (effectiveDisabled.value) {
    e?.preventDefault?.();
    return;
  }

  if (props.confirm != null) {
    // Block native submit until the user answers; cancel does not submit the form.
    e?.preventDefault?.();
    const conf = props.confirm;
    const message = typeof conf === 'string' ? conf : conf.message;
    const title = typeof conf === 'string' ? undefined : conf.title;
    const destructive = typeof conf === 'string' ? undefined : conf.destructive;
    const confirmText = typeof conf === 'string' ? undefined : conf.confirmText;
    const cancelText = typeof conf === 'string' ? undefined : conf.cancelText;
    try {
      await confirmChoyAction(message, title || 'Confirm', {
        destructive,
        confirmText,
        cancelText,
      });
    } catch {
      return;
    }
    // Re-check live (do not trust cached ACL computed; checker may flip during await).
    if (
      !!props.disabled ||
      !!props.busy ||
      offlineBlocked.value ||
      !isActionAllowedNow()
    ) {
      return;
    }
  }

  pending.value = true;
  try {
    if (typeof props.run === 'function') {
      await props.run();
    } else {
      emit('execute');
    }
  } finally {
    pending.value = false;
  }
}

defineExpose({
  onClick,
  effectiveDisabled,
  aclVisible,
  offlineBlocked,
  pending,
});
</script>
