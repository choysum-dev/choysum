// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { RowEventPayload } from '@/web';

/**
 * Build the partner detail path for a list row-click when the actor may open it.
 * Blank ids and denied access fail closed.
 */
export function resolvePartnerDetailPath(
  payload: RowEventPayload,
  canOpenDetail: boolean,
): string | null {
  if (!canOpenDetail) return null;
  const id = String(payload.row?.Id ?? '').trim();
  return id ? `/partner/partners/${id}` : null;
}

/**
 * Navigate to partner detail when the row resolves to a path.
 */
export function navigatePartnerDetail(
  payload: RowEventPayload,
  canOpenDetail: boolean,
  push: (path: string) => unknown,
): boolean {
  const path = resolvePartnerDetailPath(payload, canOpenDetail);
  if (!path) return false;
  push(path);
  return true;
}
