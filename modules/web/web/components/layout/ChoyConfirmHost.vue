<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Dialog v-model:open="open">
    <DialogContent
      class="sm:max-w-md"
      data-testid="choy-confirm-dialog"
      @pointer-down-outside="onDismiss"
      @escape-key-down="onDismiss"
    >
      <DialogTitle>{{ store.title }}</DialogTitle>
      <DialogDescription class="whitespace-pre-wrap text-sm text-foreground/80">
        {{ store.message }}
      </DialogDescription>
      <div class="mt-4 flex justify-end gap-2">
        <ChoyButton
          variant="outline"
          size="sm"
          data-testid="choy-confirm-cancel"
          @click="onCancel"
        >
          {{ store.cancelText }}
        </ChoyButton>
        <ChoyButton
          :variant="store.destructive ? 'destructive' : 'default'"
          size="sm"
          data-testid="choy-confirm-ok"
          @click="onConfirm"
        >
          {{ store.confirmText }}
        </ChoyButton>
      </div>
    </DialogContent>
  </Dialog>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import ChoyButton from '../layout/ChoyButton.vue';
import Dialog from '../vendor/ui/dialog/Dialog.vue';
import DialogContent from '../vendor/ui/dialog/DialogContent.vue';
import DialogDescription from '../vendor/ui/dialog/DialogDescription.vue';
import DialogTitle from '../vendor/ui/dialog/DialogTitle.vue';
import { resolveConfirmChoy, useConfirmChoyStore } from '../../composables/confirmChoyAction';

/**
 * App-root host for confirmChoyAction / confirmChoyChoice (Dense Admin §5.9).
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
</script>
