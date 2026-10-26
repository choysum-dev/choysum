<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <AlertDialog v-model:open="open">
    <AlertDialogContent class="sm:max-w-md" data-testid="choy-confirm-dialog">
      <AlertDialogHeader>
        <AlertDialogTitle>{{ store.title }}</AlertDialogTitle>
        <AlertDialogDescription class="whitespace-pre-wrap">
          {{ store.message }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <!--
          Capture settles the Promise before Reka AlertDialog emits update:open=false,
          so the v-model setter does not treat Action/Cancel as overlay dismiss.
        -->
        <AlertDialogCancel data-testid="choy-confirm-cancel" @click.capture="onCancel">
          {{ store.cancelText }}
        </AlertDialogCancel>
        <AlertDialogAction
          data-testid="choy-confirm-ok"
          :variant="store.destructive ? 'destructive' : 'default'"
          @click.capture="onConfirm"
        >
          {{ store.confirmText }}
        </AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import AlertDialog from '../vendor/ui/alert-dialog/AlertDialog.vue';
import AlertDialogAction from '../vendor/ui/alert-dialog/AlertDialogAction.vue';
import AlertDialogCancel from '../vendor/ui/alert-dialog/AlertDialogCancel.vue';
import AlertDialogContent from '../vendor/ui/alert-dialog/AlertDialogContent.vue';
import AlertDialogDescription from '../vendor/ui/alert-dialog/AlertDialogDescription.vue';
import AlertDialogFooter from '../vendor/ui/alert-dialog/AlertDialogFooter.vue';
import AlertDialogHeader from '../vendor/ui/alert-dialog/AlertDialogHeader.vue';
import AlertDialogTitle from '../vendor/ui/alert-dialog/AlertDialogTitle.vue';
import { resolveConfirmChoy, useConfirmChoyStore } from '../../composables/confirmChoyAction';

/**
 * App-root host for confirmChoyAction / confirmChoyChoice.
 * Uses AlertDialog chrome; Promise / store contract is unchanged.
 */
const store = useConfirmChoyStore();
const open = computed({
  get: () => store.open,
  set: (v: boolean) => {
    if (v) {
      store.open = true;
      return;
    }
    // Already settled by Action/Cancel (resolve cleared); just sync closed.
    if (!store.resolve) {
      store.open = false;
      return;
    }
    // Still pending: close UI now, but defer dismiss so a same-tick confirm/cancel
    // can clear resolve first (capture vs Reka close order is not guaranteed).
    const pending = store.resolve;
    store.open = false;
    queueMicrotask(() => {
      if (store.resolve === pending) onDismiss();
    });
  },
});

function onConfirm() {
  resolveConfirmChoy('confirm');
}

function onCancel() {
  resolveConfirmChoy('cancel');
}

function onDismiss() {
  resolveConfirmChoy(store.distinguishCancelAndClose ? 'dismiss' : 'cancel');
}

defineExpose({ onConfirm, onCancel, onDismiss, open });
</script>
