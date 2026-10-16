// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { flushPromises, mountApp } from '@/web/web/__tests__/mountApp';
import ChoyConfirmHost from './ChoyConfirmHost.vue';
import {
  confirmChoyAction,
  useConfirmChoyStore,
} from '../../composables/confirmChoyAction';

describe('ChoyConfirmHost', () => {
  test('renders destructive confirm and settles via buttons', async () => {
    const mounted = mountApp(ChoyConfirmHost as any);
    await flushPromises();
    const pending = confirmChoyAction('Delete?', 'Confirm delete', {
      confirmText: 'Delete',
      cancelText: 'Cancel',
      destructive: true,
    });
    await flushPromises();
    const store = useConfirmChoyStore();
    expect(store.open).toBe(true);
    expect(store.destructive).toBe(true);
    const ok = mounted.q('[data-testid=choy-confirm-ok]') as HTMLElement | null;
    expect(ok).not.toBeNull();
    expect(ok!.className).toContain('bg-destructive');
    expect(ok!.className).not.toContain('bg-primary');
    // Prefer setupState handlers (DOM click on nested Button is unreliable in FE harness).
    const state = mounted.setupState() as any;
    expect(typeof state?.onConfirm).toBe('function');
    state.onConfirm();
    await pending;
    expect(store.open).toBe(false);

    const cancelled = confirmChoyAction('Again?', 'Title', { destructive: false });
    await flushPromises();
    expect(store.destructive).toBe(false);
    state.onCancel();
    await cancelled.then(
      () => {
        throw new Error('expected cancel');
      },
      (e) => {
        expect(e).toBe('cancel');
      },
    );

    const dismissed = confirmChoyAction('Third?', 'Title', {
      distinguishCancelAndClose: true,
    });
    await flushPromises();
    state.onDismiss();
    await dismissed.then(
      () => {
        throw new Error('expected dismiss');
      },
      (e) => {
        expect(e).toBe('dismiss');
      },
    );

    // Cover v-model:open setter → onDismiss when distinguishing is off (maps to cancel).
    const viaOpen = confirmChoyAction('Fourth?', 'Title');
    await flushPromises();
    expect(store.open).toBe(true);
    // Writable computed `open` is exposed on setupState; assigning false runs onDismiss.
    state.open = false;
    await viaOpen.then(
      () => {
        throw new Error('expected cancel');
      },
      (e) => {
        expect(e).toBe('cancel');
      },
    );
    expect(store.open).toBe(false);
    mounted.unmount();
  });

  test('Action close after onConfirm does not settle as cancel', async () => {
    const mounted = mountApp(ChoyConfirmHost as any);
    await flushPromises();
    const pending = confirmChoyAction('Delete?', 'Confirm delete', { destructive: true });
    await flushPromises();
    const store = useConfirmChoyStore();
    const state = mounted.setupState() as any;
    // Capture path: settle confirm first, then AlertDialog emits update:open=false.
    state.onConfirm();
    state.open = false;
    await pending;
    expect(store.open).toBe(false);
    expect(store.resolve).toBeNull();
    mounted.unmount();
  });

  test('open=false while resolve is pending still dismisses as cancel', async () => {
    const mounted = mountApp(ChoyConfirmHost as any);
    await flushPromises();
    const pending = confirmChoyAction('Leave?', 'Title');
    await flushPromises();
    const state = mounted.setupState() as any;
    state.open = false;
    await pending.then(
      () => {
        throw new Error('expected cancel');
      },
      (e) => {
        expect(e).toBe('cancel');
      },
    );
    mounted.unmount();
  });
});
