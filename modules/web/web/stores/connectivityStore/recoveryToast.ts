// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { watch } from 'vue';
import { getActivePinia } from 'pinia';
import { createTranslate } from '@/web/web/i18n';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { useConnectivityStore } from './index';

const { _t } = createTranslate('web', { scope: 'web/stores/connectivityStore' });

let stopWatch: (() => void) | undefined;
let lastRecoveryGen = 0;

/**
 * Once per unreachable→reachable edge, show a single recovery info toast.
 * Call after Pinia is active (with bindConnectivityLifecycle).
 */
export function bindConnectivityRecoveryToast(): void {
  if (!getActivePinia()) return;
  stopWatch?.();
  const store = useConnectivityStore();
  lastRecoveryGen = store.recoveryGeneration;
  stopWatch = watch(
    () => store.recoveryGeneration,
    gen => {
      if (gen <= lastRecoveryGen) return;
      lastRecoveryGen = gen;
      ChoyMessage.info(_t('Connection restored'));
    }
  );
}

export function unbindConnectivityRecoveryToastForTests(): void {
  stopWatch?.();
  stopWatch = undefined;
  lastRecoveryGen = 0;
}
