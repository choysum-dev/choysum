// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { confirmChoyAction, confirmChoyChoice, resolveConfirmChoy, useConfirmChoyStore } from './confirmChoyAction.ts';

describe('confirmChoyAction', () => {
  test('resolves on confirm and rejects on dismiss', async () => {
    const store = useConfirmChoyStore();
    const pending = confirmChoyAction('Delete this?', 'Confirm delete');
    expect(store.open).toBe(true);
    expect(store.message).toBe('Delete this?');
    resolveConfirmChoy('confirm');
    await pending;
    expect(store.open).toBe(false);

    const rejected = confirmChoyAction('Again?', 'Title');
    resolveConfirmChoy('dismiss');
    await rejected.then(() => { throw new Error('expected reject'); }, (e) => { expect(e).toBe('dismiss'); });
  });

  test('confirmChoyChoice returns raw choice for save/discard flows', async () => {
    const pending = confirmChoyChoice('Unsaved?', 'Unsaved changes', {
      confirmText: 'Save',
      cancelText: 'Discard',
      distinguishCancelAndClose: true,
    });
    resolveConfirmChoy('cancel');
    expect(await pending).toBe('cancel');
  });

  test('sets destructive flag for irreversible confirms', async () => {
    const store = useConfirmChoyStore();
    const pending = confirmChoyAction('Delete?', 'Confirm delete', { destructive: true });
    expect(store.destructive).toBe(true);
    resolveConfirmChoy('confirm');
    await pending;
  });
});
