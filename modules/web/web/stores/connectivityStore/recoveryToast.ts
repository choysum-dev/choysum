// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { watch } from 'vue';
import { getActivePinia } from 'pinia';
import { createTranslate } from '@/web/web/i18n';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { useConnectivityStore } from './index';

const { _t } = createTranslate('web', { scope: 'web/stores/connectivityStore' });

let stopWatch: (() => void) | undefined;
/** Last generation that already produced a toast (−1 = none yet). */
let lastNotifiedGen = -1;

/**
 * Once per unreachable→reachable edge, show a single recovery info toast.
 * Call after Pinia is active (with bindConnectivityLifecycle).
 */
export function bindConnectivityRecoveryToast(): void {
  if (!getActivePinia()) return;
  stopWatch?.();
  const store = useConnectivityStore();
  stopWatch = watch(
    () => store.recoveryGeneration,
    gen => {
      // gen 0 is the idle baseline — never toast it.
      if (gen <= 0 || gen <= lastNotifiedGen) return;
      // Session may have flipped unreachable again before this watcher bound.
      if (store.status !== 'reachable') return;
      lastNotifiedGen = gen;
      ChoyMessage.info(_t('Connection restored'));
    },
    // Catch a recovery that landed before this watcher bound.
    { immediate: true },
  );
}

export function unbindConnectivityRecoveryToastForTests(): void {
  stopWatch?.();
  stopWatch = undefined;
  lastNotifiedGen = -1;
}
