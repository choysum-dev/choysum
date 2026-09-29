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

  test('opening a second confirm settles the previous pending promise', async () => {
    const first = confirmChoyAction('First?', 'One');
    const second = confirmChoyAction('Second?', 'Two');
    await first.then(
      () => {
        throw new Error('expected first to be superseded');
      },
      (e) => {
        expect(e).toBe('dismiss');
      },
    );
    resolveConfirmChoy('confirm');
    await second;
  });
});
