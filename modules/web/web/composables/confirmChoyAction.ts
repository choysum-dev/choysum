// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { reactive } from 'vue';

export type ConfirmChoyChoice = 'confirm' | 'cancel' | 'dismiss';

export type ConfirmChoyOptions = {
  confirmText?: string;
  cancelText?: string;
  /** When true, Cancel resolves as 'cancel' and overlay/X as 'dismiss'. */
  distinguishCancelAndClose?: boolean;
};

type ConfirmRequest = {
  open: boolean;
  title: string;
  message: string;
  confirmText: string;
  cancelText: string;
  distinguishCancelAndClose: boolean;
  resolve: ((choice: ConfirmChoyChoice) => void) | null;
};

const state: ConfirmRequest = reactive({
  open: false,
  title: '',
  message: '',
  confirmText: 'OK',
  cancelText: 'Cancel',
  distinguishCancelAndClose: false,
  resolve: null,
});

/** Reactive confirm dialog state for the App-mounted host. */
export function useConfirmChoyStore() {
  return state;
}

function finish(choice: ConfirmChoyChoice) {
  const resolve = state.resolve;
  state.open = false;
  state.resolve = null;
  resolve?.(choice);
}

export function resolveConfirmChoy(choice: ConfirmChoyChoice) {
  finish(choice);
}

/** Reject/settle any in-flight confirm before opening another (avoids leaked promises). */
function supersedePending(choice: ConfirmChoyChoice = 'dismiss'): void {
  if (!state.resolve) return;
  finish(choice);
}

/**
 * Promise confirm dialog. Resolves on confirm; rejects with 'cancel' or 'dismiss'.
 * Host must mount `<ChoyConfirmHost />` (App root).
 */
export function confirmChoyAction(
  message: string,
  title = 'Confirm',
  options?: ConfirmChoyOptions
): Promise<void> {
  return new Promise((resolve, reject) => {
    supersedePending('dismiss');
    state.title = title;
    state.message = message;
    state.confirmText = options?.confirmText || 'OK';
    state.cancelText = options?.cancelText || 'Cancel';
    state.distinguishCancelAndClose = Boolean(options?.distinguishCancelAndClose);
    state.open = true;
    state.resolve = (choice: ConfirmChoyChoice) => {
      if (choice === 'confirm') resolve();
      else reject(choice);
    };
  });
}

/**
 * Same dialog as confirmChoyAction, but returns the raw choice (for save/discard/cancel flows).
 */
export function confirmChoyChoice(
  message: string,
  title = 'Confirm',
  options?: ConfirmChoyOptions
): Promise<ConfirmChoyChoice> {
  return new Promise(resolve => {
    supersedePending('dismiss');
    state.title = title;
    state.message = message;
    state.confirmText = options?.confirmText || 'OK';
    state.cancelText = options?.cancelText || 'Cancel';
    state.distinguishCancelAndClose = Boolean(options?.distinguishCancelAndClose);
    state.open = true;
    state.resolve = resolve;
  });
}
