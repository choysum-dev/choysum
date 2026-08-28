// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed } from 'vue';
import { getCurrentRequestContext } from '@/core/rpc/context';
import { normalizeExportFieldPaths } from '@/core/web/export/field_paths';
import { buildUnifiedQuery } from '@/web/web/query/context';
import { exportFieldSelection } from '@/web/web/query/utils/registry/field';

export type RecordExportListRef = {
  selectedItems?: { value?: Array<{ Id?: string }> } | Array<{ Id?: string }> | null;
} | null;

export type UseRecordExportScopeOptions = {
  store: { storeId?: string; state?: { result?: { total?: number } } };
  getListRef: () => RecordExportListRef;
};

function collectSelectedIds(listRef: RecordExportListRef): string[] {
  const raw = listRef?.selectedItems;
  if (Array.isArray(raw)) {
    return raw.map(row => String(row?.Id ?? '').trim()).filter(Boolean);
  }
  const items = raw?.value ?? [];
  if (!Array.isArray(items)) {
    return [];
  }
  return items.map(row => String(row?.Id ?? '').trim()).filter(Boolean);
}

/**
 * Derives ExportPanel scope (ids / domain / defaultFields / filteredCount) from list + store.
 */
export function useRecordExportScope(options: UseRecordExportScopeOptions) {
  const companyId = computed(() => {
    const ctx = getCurrentRequestContext();
    return String(ctx?.activeCompanyId ?? ctx?.companyId ?? '').trim();
  });

  const ids = computed(() => collectSelectedIds(options.getListRef()));

  const domain = computed(() => {
    const ctx = buildUnifiedQuery(options.store as any, {
      execOptions: { skipPagination: true, skipCount: true },
    });
    return JSON.stringify(ctx.filters ?? { And: [] });
  });

  const defaultFields = computed(() => {
    const storeId = String(options.store.storeId ?? '');
    const paths = exportFieldSelection(storeId) ?? [];
    return normalizeExportFieldPaths(paths.filter(path => path !== 'Id'));
  });

  const filteredCount = computed(() =>
    Number((options.store.state as { result?: { total?: number } } | undefined)?.result?.total ?? 0)
  );

  return { companyId, ids, domain, defaultFields, filteredCount };
}
