// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue';
import { companyScopeViewKey } from '../components/layout/companyScopeViewKey';

/**
 * Auth publishes active/enabled company ids here so the web shell can remount
 * views without importing the auth store.
 */
const published = ref<{ activeCompanyId: string; enabledCompanyIds: string[] }>({
  activeCompanyId: '',
  enabledCompanyIds: [],
});

/**
 * Publish the current session company scope for shell view remounts.
 */
export function publishCompanyScopeView(activeCompanyId: unknown, enabledCompanyIds: unknown): void {
  const active = String(activeCompanyId ?? '').trim();
  const enabled = Array.isArray(enabledCompanyIds)
    ? enabledCompanyIds.map(x => String(x ?? '').trim()).filter(Boolean)
    : [];
  published.value = { activeCompanyId: active, enabledCompanyIds: enabled };
}

/**
 * Clear published company scope (logout / anonymous shell).
 */
export function clearCompanyScopeView(): void {
  published.value = { activeCompanyId: '', enabledCompanyIds: [] };
}

/**
 * Stable router-view remount key from the published company scope.
 */
export function useCompanyScopeViewKey() {
  return computed(() =>
    companyScopeViewKey(published.value.activeCompanyId, published.value.enabledCompanyIds),
  );
}
