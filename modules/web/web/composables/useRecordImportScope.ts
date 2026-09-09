// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { computed, toValue, type MaybeRefOrGetter, type Ref } from 'vue';
import { getCurrentRequestContext as defaultGetCurrentRequestContext } from '@/core/rpc/context';
import type { RecordIoConfig } from './recordIoTypes';

export type UseRecordImportScopeOptions = {
  /** Fully qualified model name (e.g. partner.Partner). */
  model?: MaybeRefOrGetter<string | null | undefined>;
  config?: RecordIoConfig | Ref<RecordIoConfig | null | undefined> | MaybeRefOrGetter<RecordIoConfig | null | undefined>;
  /** Optional override for request context (tests inject a stub). */
  getCurrentRequestContext?: typeof defaultGetCurrentRequestContext;
};

/**
 * Resolves ImportPanel props from model, RecordIoConfig, and request context.
 */
export function useRecordImportScope(options: UseRecordImportScopeOptions) {
  const getCtx = options.getCurrentRequestContext ?? defaultGetCurrentRequestContext;
  const config = computed(() => toValue(options.config) ?? null);

  const model = computed(() => String(toValue(options.model) ?? '').trim());
  const companyId = computed(() => {
    const ctx = getCtx();
    return String(ctx?.activeCompanyId ?? ctx?.companyId ?? '').trim();
  });
  const columnMapping = computed(() => config.value?.import?.columnMapping ?? {});
  const uploadHint = computed(() => config.value?.import?.uploadHint);

  return { model, companyId, columnMapping, uploadHint };
}
