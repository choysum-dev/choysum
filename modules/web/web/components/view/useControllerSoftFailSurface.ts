// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Soft-fail Retry/Cancel for collection controllers that expose
 * `retryPending` / `restoreCommitted` (ListController, Kanban engine.controller).
 * Form / Chart keep their own surface handlers.
 */

export type SoftFailController = {
  retryPending?: () => Promise<unknown>;
  restoreCommitted?: () => void;
  vm: { error: unknown };
};

export type UseControllerSoftFailSurfaceOptions = {
  getController: () => SoftFailController;
  /** Full reload when there is no pending failed intent (idle Retry). */
  reload: () => Promise<void>;
  /** Extra cancel side effects (e.g. List resets lastSearchPayload). */
  onCancelExtra?: () => void;
};

export function useControllerSoftFailSurface(opts: UseControllerSoftFailSurfaceOptions): {
  onSurfaceRetry: () => Promise<void>;
  onSurfaceCancel: () => void;
} {
  async function onSurfaceRetry(): Promise<void> {
    try {
      const ctrl = opts.getController();
      if (ctrl.vm.error != null && typeof ctrl.retryPending === 'function') {
        await ctrl.retryPending();
        return;
      }
      await opts.reload();
    } catch (err) {
      // Controllers normally set vm.error; surface a fallback if they did not.
      try {
        const ctrl = opts.getController();
        if (ctrl.vm.error == null) ctrl.vm.error = err;
      } catch {
        // Controller unavailable; keep soft-fail from becoming an unhandled rejection.
      }
    }
  }

  function onSurfaceCancel(): void {
    const ctrl = opts.getController();
    if (typeof ctrl.restoreCommitted === 'function') {
      ctrl.restoreCommitted();
    } else {
      ctrl.vm.error = null;
    }
    opts.onCancelExtra?.();
  }

  return { onSurfaceRetry, onSurfaceCancel };
}
