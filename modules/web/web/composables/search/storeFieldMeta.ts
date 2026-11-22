// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { WebFieldMetadata } from '@/web/web/stores/modelStore';

/**
 * Static field meta from `store.metadata.fields` (Search filter editor / standalone binding).
 */
export function readStoreFieldMeta(
  store: { metadata?: { fields?: Record<string, unknown> } } | null | undefined,
  fieldName: string
): Partial<WebFieldMetadata> {
  const fields = store?.metadata?.fields;
  if (!fields || !fieldName) return {};
  return (fields[fieldName] || {}) as Partial<WebFieldMetadata>;
}

/**
 * Build filter-condition field meta: static store fields overlaid with a type fallback.
 */
export function buildFilterConditionFieldMeta(
  store: { metadata?: { fields?: Record<string, unknown> } } | null | undefined,
  fieldName: string,
  typeFallback: unknown
): Partial<WebFieldMetadata> {
  const staticMeta = readStoreFieldMeta(store, fieldName);
  return {
    ...staticMeta,
    type: (staticMeta.type || typeFallback) as any,
    isReadonly: false,
  };
}
