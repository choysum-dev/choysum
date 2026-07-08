// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId } from '@/core/service/utils/normalization';

/**
 * Resolve a ManyToOne / reference field value to its canonical string ID.
 * Delegates to core's {@link normalizeRefId}.
 */
export const asRefId = normalizeRefId;

/**
 * Resolve the company scope key from a CompanyId reference.
 * Returns '__GLOBAL__' when the company id is empty/null/undefined.
 */
export function normalizeCompanyScopeKey(companyId: any): string {
  const id = asRefId(companyId);
  return id || '__GLOBAL__';
}
