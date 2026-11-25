// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, nextTick, type MaybeRefOrGetter, type Ref, toValue } from 'vue';
import { confirmChoyAction } from '@/web/web/composables/confirmChoyAction';

/** FormView API slice used by dialog footers (submit / cancel after dirty confirm). */
export type ChoyFormDialogViewApi = {
  submit?: () => Promise<unknown>;
  cancel?: () => void | Promise<void>;
  /** True when the embedded form draft differs from its baseline. */
  isDirty?: () => boolean;
  getFormData?: () => unknown;
};

/** Chrome props for embedded FormView inside ChoyDialog (no page header/actions/messages). */
export const CHOY_FORM_DIALOG_EMBEDDED_CHROME = {
  embedded: true,
  showHeader: false,
  showActions: false,
  showMessages: false,
  resolveRecordIdFromRoute: false,
} as const;

export type UseChoyFormDialogShellOptions = {
  /** Close the dialog (sets v-model open false). Bound as FormView cancel-handler after dirty confirm. */
  onClose: () => void;
  /** Ref to FormView expose (or child submit API with submit/cancel). */
  formViewRef: Ref<ChoyFormDialogViewApi | null | undefined>;
  /** When true, Esc/outside dismiss is blocked (typically while saving). */
  saving?: MaybeRefOrGetter<boolean>;
  /**
   * Extra dirty outside FormView draft (e.g. Properties selection editor).
   * When true, footer Cancel confirms once then closes without a second FormView prompt.
   */
  externalDirty?: () => boolean;
  /** Override discard confirm for externalDirty (defaults to standard unsaved-changes copy). */
  confirmExternalDiscard?: () => Promise<boolean>;
  /**
   * When true, footer Cancel skips FormView.cancel and calls onClose directly
   * (readonly Close, or hosts that intentionally hard-close).
   */
  skipFormCancel?: MaybeRefOrGetter<boolean>;
};

async function defaultConfirmExternalDiscard(): Promise<boolean> {
  try {
    await confirmChoyAction(
      'You have unsaved changes. Discard them?',
      'Unsaved changes',
      {
        confirmText: 'Discard',
        cancelText: 'Cancel',
        destructive: true,
      },
    );
    return true;
  } catch {
    return false;
  }
}

/**
 * Shared Form-in-Dialog contract: embedded chrome props, cancelHandler → close,
 * footer Save → form.submit(), footer Cancel → external dirty confirm or form.cancel().
 */
export function useChoyFormDialogShell(opts: UseChoyFormDialogShellOptions) {
  const dismissible = computed(() => !toValue(opts.saving ?? false));

  function cancelHandler(): void {
    opts.onClose();
  }

  async function onFooterSave(): Promise<void> {
    const api = opts.formViewRef.value;
    if (api?.submit) {
      await api.submit();
      return;
    }
    // Ref not yet populated (form still mounting); retry once after the next tick.
    await nextTick();
    const retry = opts.formViewRef.value;
    if (retry?.submit) {
      await retry.submit();
    }
  }

  async function onFooterCancel(): Promise<void> {
    if (toValue(opts.skipFormCancel ?? false)) {
      opts.onClose();
      return;
    }
    if (opts.externalDirty?.()) {
      const confirm = opts.confirmExternalDiscard ?? defaultConfirmExternalDiscard;
      if (!(await confirm())) return;
      opts.onClose();
      return;
    }
    const api = opts.formViewRef.value;
    if (api?.cancel) {
      await api.cancel();
      return;
    }
    opts.onClose();
  }

  return {
    /** Spread onto ChoyFormView (`v-bind="embeddedFormChrome"`). */
    embeddedFormChrome: CHOY_FORM_DIALOG_EMBEDDED_CHROME,
    /** Bind to FormView `:cancel-handler`. */
    cancelHandler,
    /** Footer Save: programmatic submit (footer outside `<form>`). */
    onFooterSave,
    /** Footer Cancel: dirty confirm then close. */
    onFooterCancel,
    /** Bind to ChoyDialogContent `:dismissible`. */
    dismissible,
  };
}
