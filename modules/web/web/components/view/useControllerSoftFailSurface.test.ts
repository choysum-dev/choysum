// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { fnRecorder } from '@/web/web/__tests__/mountApp';
import { useControllerSoftFailSurface } from './useControllerSoftFailSurface';

describe('useControllerSoftFailSurface', () => {
  test('retryPending when error is set; otherwise reload', async () => {
    const retryPending = fnRecorder(async () => {});
    const reload = fnRecorder(async () => {});
    const ctrl = {
      retryPending: () => retryPending(),
      vm: { error: new Error('x') as unknown },
    };
    const { onSurfaceRetry } = useControllerSoftFailSurface({
      getController: () => ctrl,
      reload: () => reload(),
    });
    await onSurfaceRetry();
    expect(retryPending.calls.length).toBe(1);
    expect(reload.calls.length).toBe(0);

    ctrl.vm.error = null;
    await onSurfaceRetry();
    expect(reload.calls.length).toBe(1);
  });

  test('cancel prefers restoreCommitted then onCancelExtra', () => {
    const restoreCommitted = fnRecorder(() => {});
    const onCancelExtra = fnRecorder(() => {});
    const ctrl = {
      restoreCommitted: () => restoreCommitted(),
      vm: { error: new Error('x') as unknown },
    };
    const { onSurfaceCancel } = useControllerSoftFailSurface({
      getController: () => ctrl,
      reload: async () => {},
      onCancelExtra: () => onCancelExtra(),
    });
    onSurfaceCancel();
    expect(restoreCommitted.calls.length).toBe(1);
    expect(onCancelExtra.calls.length).toBe(1);
    expect(ctrl.vm.error).not.toBeNull();
  });

  test('cancel clears vm.error when restoreCommitted is missing', () => {
    const ctrl = { vm: { error: new Error('x') as unknown } };
    const { onSurfaceCancel } = useControllerSoftFailSurface({
      getController: () => ctrl,
      reload: async () => {},
    });
    onSurfaceCancel();
    expect(ctrl.vm.error).toBeNull();
  });

  test('retry swallows thrown errors from retryPending', async () => {
    const ctrl = {
      retryPending: async () => {
        throw new Error('offline');
      },
      vm: { error: new Error('x') as unknown },
    };
    const { onSurfaceRetry } = useControllerSoftFailSurface({
      getController: () => ctrl,
      reload: async () => {},
    });
    let threw = false;
    try {
      await onSurfaceRetry();
    } catch {
      threw = true;
    }
    expect(threw).toBe(false);
  });
});
