// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

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

  try {
    const menuStore = useMenuStore();
    watch(
      () => menuStore.activeMenuId,
      (id) => {
        stickyMenuId.value = stickyLeafMenuId(stickyMenuId.value, id);
      },
      { immediate: true },
    );
  } catch {
    // Guest / no Pinia menu store — company key only (empty menu fragment).
  }

  return computed(() => keepAliveScopeKey(companyKey.value, stickyMenuId.value));
}
