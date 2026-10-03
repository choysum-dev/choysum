<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <AlertDialog v-model:open="open">
    <AlertDialogContent class="sm:max-w-md" data-testid="choy-confirm-dialog">
      <AlertDialogHeader>
        <AlertDialogTitle>{{ store.title }}</AlertDialogTitle>
        <AlertDialogDescription class="whitespace-pre-wrap text-sm text-foreground/80">
          {{ store.message }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel data-testid="choy-confirm-cancel" @click="onCancel">
          {{ store.cancelText }}
        </AlertDialogCancel>
        <AlertDialogAction
          data-testid="choy-confirm-ok"
          :class="store.destructive ? buttonVariants({ variant: 'destructive', size: 'sm' }) : buttonVariants({ size: 'sm' })"
          @click="onConfirm"
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
import { buttonVariants } from '../vendor/ui/button';
import { resolveConfirmChoy, useConfirmChoyStore } from '../../composables/confirmChoyAction';

/**
 * App-root host for confirmChoyAction / confirmChoyChoice.
 * Uses AlertDialog chrome; Promise / store contract is unchanged.
 */
const store = useConfirmChoyStore();
const open = computed({
  get: () => store.open,
  set: (v: boolean) => {
    if (!v && store.open) onDismiss();
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
