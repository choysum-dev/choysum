<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <AlertDialog v-model:open="open">
    <!--
      Reka may still stamp dir=ltr on the portaled node; inline direction +
      physical text/footer alignment follow confirmDir so RTL cannot lose to kit
      sm:text-left / default useDirection.
    -->
    <AlertDialogContent
      class="sm:max-w-md"
      data-testid="choy-confirm-dialog"
      :dir="confirmDir"
      :style="contentStyle"
    >
      <AlertDialogHeader class="text-start" :style="headerStyle">
        <AlertDialogTitle :style="copyStyle">{{ store.title }}</AlertDialogTitle>
        <AlertDialogDescription class="whitespace-pre-wrap" :style="copyStyle">
          {{ store.message }}
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter :style="footerStyle">
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
import { useI18nStore } from '../../stores/i18nStore';

/**
 * App-root host for confirmChoyAction / confirmChoyChoice.
 * Uses AlertDialog chrome; Promise / store contract is unchanged.
 */
const store = useConfirmChoyStore();

const confirmDir = computed<'ltr' | 'rtl'>(() => {
  try {
    return useI18nStore().activeLanguage.textDirection === 'rtl' ? 'rtl' : 'ltr';
  } catch {
    return 'ltr';
  }
});

/** Force used direction even if Reka's dir attr stays ltr. */
const contentStyle = computed(() => ({
  direction: confirmDir.value,
}));

/** Physical align: kit sm:text-left / CSS start+plaintext otherwise wins under RTL. */
const headerStyle = computed(() => ({
  textAlign: confirmDir.value === 'rtl' ? 'right' : 'left',
}));

/**
 * Plaintext keeps English punctuation (trailing ?) from mirroring under
 * direction:rtl; pair with physical text-align, not start.
 */
const copyStyle = computed(() => ({
  textAlign: confirmDir.value === 'rtl' ? 'right' : 'left',
  unicodeBidi: 'plaintext' as const,
}));

/**
 * Keep actions on the same physical side as form toolbars (inline-start of the
 * page chrome): right in both LTR (flex-end) and RTL (flex-start).
 */
const footerStyle = computed(() => ({
  justifyContent: confirmDir.value === 'rtl' ? 'flex-start' : 'flex-end',
}));
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
