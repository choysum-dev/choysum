// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { resolveListRowRecordId } from '@/base/web/views/list_row_nav';

/**
 * Build the partner detail path for a list row-click when the actor may open it.
 * Accepts a raw row or `{ row }` wrapper; blank ids and denied access fail closed.
 */
export function resolvePartnerDetailPath(
  payload: unknown,
  canOpenDetail: boolean,
): string | null {
  if (!canOpenDetail) return null;
  const id = resolveListRowRecordId(payload);
  return id ? `/partner/partners/${id}` : null;
}

/**
 * Navigate to partner detail when the row resolves to a path.
 */
export function navigatePartnerDetail(
  payload: unknown,
  canOpenDetail: boolean,
  push: (path: string) => unknown,
): boolean {
  const path = resolvePartnerDetailPath(payload, canOpenDetail);
  if (!path) return false;
  push(path);
  return true;
}
