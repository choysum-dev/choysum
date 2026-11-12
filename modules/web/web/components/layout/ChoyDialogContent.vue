<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import VendorDialogContent from '../vendor/ui/dialog/DialogContent.vue';
import type { ClassValue } from '../../lib/utils';
import { lockDialogDismiss } from './choyDialogDismiss';

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    /** When false, omit the corner close control. */
    showCloseButton?: boolean;
    /**
     * When false, Esc and pointer-outside dismiss are prevented
     * (e.g. while a submit is in flight).
     */
    dismissible?: boolean;
  }>(),
  {
    showCloseButton: true,
    dismissible: true,
  },
);

defineOptions({ inheritAttrs: false });

function onDismissAttempt(event: Event): void {
  lockDialogDismiss(event, props.dismissible);
}
</script>

<template>
  <VendorDialogContent
    v-bind="$attrs"
    :class="props.class"
    :show-close-button="showCloseButton"
    @escape-key-down="onDismissAttempt"
    @pointer-down-outside="onDismissAttempt"
    @focus-outside="onDismissAttempt"
    @interact-outside="onDismissAttempt"
  >
    <slot />
  </VendorDialogContent>
</template>
