// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { nextTick } from 'vue';

/**
 * useCancelableEmit
 * A tiny helper to emit cancellable events with a confirm/cancel contract.
 * - If the consumer doesn't call confirm/cancel synchronously, it will auto-confirm on nextTick.
 */
export function useCancelableEmit<
  // eslint-disable-next-line @typescript-eslint/ban-types
  TEmit extends (event: string, ...args: any[]) => void
>(emit: TEmit) {
  async function emitCancelable<T extends object = Record<string, unknown>>(name: string, data?: T): Promise<boolean> {
    let settled = false;
    return new Promise(async resolve => {
      const confirm = () => {
        if (!settled) {
          settled = true;
          resolve(true);
        }
      };
      const cancel = () => {
        if (!settled) {
          settled = true;
          resolve(false);
        }
      };
      emit(name, Object.assign({}, data || {}, { confirm, cancel }));
      await nextTick();
      if (!settled) resolve(true);
    });
  }

  return { emitCancelable };
}
