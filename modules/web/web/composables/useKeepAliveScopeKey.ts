// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getActivePinia } from 'pinia';
import { computed, ref, watch } from 'vue';
import {
  keepAliveScopeKey,
  stickyLeafMenuId,
} from '../components/layout/keepAliveScopeKey';
import { useCompanyScopeViewKey } from '../stores/companyScopeView';
import { useMenuStore } from '../stores/menuStore';

/**
 * App-shell KeepAlive remount key: company scope + sticky leaf menu id.
 * Ignores brief activeMenuId → null flaps so cold-start sync does not drop cache.
 */
export function useKeepAliveScopeKey() {
  const companyKey = useCompanyScopeViewKey();
  const stickyMenuId = ref('');

  // No active Pinia (Guest / early mount) → company key only; do not swallow store bugs.
  if (getActivePinia()) {
    const menuStore = useMenuStore();
    watch(
      () => menuStore.activeMenuId,
      (id) => {
        stickyMenuId.value = stickyLeafMenuId(stickyMenuId.value, id);
      },
      { immediate: true },
    );
  }

  return computed(() => keepAliveScopeKey(companyKey.value, stickyMenuId.value));
}
