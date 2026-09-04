// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeRefId as normalizeRefIdCore } from '@/core/service/utils/normalization';

/** Normalize a tree/node ref value, unwrapping entity proxies when present. */
export function normalizeTreeRefId(v: unknown): string | null {
  if (v != null && typeof v === 'object' && typeof (v as { toEntity?: () => unknown }).toEntity === 'function') {
    return normalizeRefIdCore((v as { toEntity: () => unknown }).toEntity());
  }
  return normalizeRefIdCore(v);
}
