// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId } from '@/core/service/utils/normalization';

/**
 * Resolve the company scope key from a CompanyId reference.
 * Returns '__GLOBAL__' when the company id is empty/null/undefined.
 */
export function normalizeCompanyScopeKey(companyId: any): string {
  const id = normalizeRefId(companyId);
  return id || '__GLOBAL__';
}
