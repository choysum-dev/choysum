// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue';
import { resolveConfirmChoy } from './confirmChoyAction';
import {
  CHOY_FORM_DIALOG_EMBEDDED_CHROME,
  useChoyFormDialogShell,
  type ChoyFormDialogViewApi,
} from './useChoyFormDialogShell';

describe('useChoyFormDialogShell', () => {
  test('embeddedFormChrome matches dialog FormView contract', () => {
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const shell = useChoyFormDialogShell({
      onClose: () => {},
      formViewRef,
    });
    expect(shell.embeddedFormChrome).toEqual(CHOY_FORM_DIALOG_EMBEDDED_CHROME);
    expect(shell.embeddedFormChrome.embedded).toBe(true);
    expect(shell.embeddedFormChrome.showHeader).toBe(false);
    expect(shell.embeddedFormChrome.showActions).toBe(false);
    expect(shell.embeddedFormChrome.showMessages).toBe(false);
    expect(shell.embeddedFormChrome.resolveRecordIdFromRoute).toBe(false);
  });

  test('dismissible is false while saving', () => {
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const saving = ref(false);
    const shell = useChoyFormDialogShell({
      onClose: () => {},
      formViewRef,
      saving,
    });
    expect(shell.dismissible.value).toBe(true);
    saving.value = true;
    expect(shell.dismissible.value).toBe(false);
  });

  test('cancelHandler closes via onClose', () => {
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    let closed = 0;
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
    });
    shell.cancelHandler();
    expect(closed).toBe(1);
  });

  test('onFooterSave calls formView.submit', async () => {
    let submitCalls = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>({
      submit: async () => {
        submitCalls += 1;
        return { ok: true };
      },
    });
    const shell = useChoyFormDialogShell({
      onClose: () => {},
      formViewRef,
    });
    await shell.onFooterSave();
    expect(submitCalls).toBe(1);
  });

  test('onFooterSave is a no-op without submit', async () => {
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const shell = useChoyFormDialogShell({
      onClose: () => {},
      formViewRef,
    });
    await shell.onFooterSave();
  });

  test('onFooterSave retries submit after nextTick when ref populates late', async () => {
    let submitCalls = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const shell = useChoyFormDialogShell({
      onClose: () => {},
      formViewRef,
    });
    const pending = shell.onFooterSave();
    formViewRef.value = {
      submit: async () => {
        submitCalls += 1;
        return { ok: true };
      },
    };
    await pending;
    expect(submitCalls).toBe(1);
  });

  test('onFooterCancel uses formView.cancel when present', async () => {
    let cancelCalls = 0;
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>({
      cancel: async () => {
        cancelCalls += 1;
      },
    });
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
    });
    await shell.onFooterCancel();
    expect(cancelCalls).toBe(1);
    expect(closed).toBe(0);
  });

  test('onFooterCancel keeps dialog open when formView.cancel rejects', async () => {
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>({
      cancel: async () => {
        throw new Error('discard dismissed');
      },
    });
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
    });
    await shell.onFooterCancel();
    expect(closed).toBe(0);
  });

  test('onFooterCancel falls back to onClose without cancel API', async () => {
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
    });
    await shell.onFooterCancel();
    expect(closed).toBe(1);
  });

  test('onFooterCancel skips form cancel when skipFormCancel', async () => {
    let cancelCalls = 0;
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>({
      cancel: async () => {
        cancelCalls += 1;
      },
    });
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
      skipFormCancel: true,
    });
    await shell.onFooterCancel();
    expect(cancelCalls).toBe(0);
    expect(closed).toBe(1);
  });

  test('onFooterCancel confirms external dirty then closes without form.cancel', async () => {
    let cancelCalls = 0;
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>({
      cancel: async () => {
        cancelCalls += 1;
      },
    });
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
      externalDirty: () => true,
    });
    const pending = shell.onFooterCancel();
    resolveConfirmChoy('confirm');
    await pending;
    expect(cancelCalls).toBe(0);
    expect(closed).toBe(1);
  });

  test('onFooterCancel aborts when external discard is rejected', async () => {
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
      externalDirty: () => true,
      confirmExternalDiscard: async () => false,
    });
    await shell.onFooterCancel();
    expect(closed).toBe(0);
  });

  test('custom confirmExternalDiscard is preferred over default', async () => {
    let customCalls = 0;
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
      externalDirty: () => true,
      confirmExternalDiscard: async () => {
        customCalls += 1;
        return true;
      },
    });
    await shell.onFooterCancel();
    expect(customCalls).toBe(1);
    expect(closed).toBe(1);
  });

  test('default external discard rejects on dismiss', async () => {
    let closed = 0;
    const formViewRef = ref<ChoyFormDialogViewApi | null>(null);
    const shell = useChoyFormDialogShell({
      onClose: () => {
        closed += 1;
      },
      formViewRef,
      externalDirty: () => true,
    });
    const pending = shell.onFooterCancel();
    resolveConfirmChoy('dismiss');
    await pending;
    expect(closed).toBe(0);
  });
});
