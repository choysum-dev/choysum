// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, toValue, type Ref } from 'vue';
import { getCurrentRequestContext } from '@/core/rpc/context';
import type { RecordIoConfig } from './recordIoTypes';

export type UseRecordImportScopeOptions = {
  config: RecordIoConfig | Ref<RecordIoConfig>;
};

/**
 * Resolves ImportPanel props from RecordIoConfig and request context.
 */
export function useRecordImportScope(options: UseRecordImportScopeOptions) {
  const config = computed(() => toValue(options.config));

  const model = computed(() => config.value.model);
  const companyId = computed(() => {
    const ctx = getCurrentRequestContext();
    return String(ctx?.activeCompanyId ?? ctx?.companyId ?? '').trim();
  });
  const columnMapping = computed(() => config.value.import?.columnMapping ?? {});
  const uploadHint = computed(() => config.value.import?.uploadHint);

  return { model, companyId, columnMapping, uploadHint };
}
