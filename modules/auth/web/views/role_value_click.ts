// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a Role Id from an OManyToOneField value-click payload.
 * Empty / whitespace → '' (caller should no-op).
 */
export function roleIdFromValueClick(payload: { id?: unknown } | null | undefined): string {
  return String(payload?.id || '').trim();
}
