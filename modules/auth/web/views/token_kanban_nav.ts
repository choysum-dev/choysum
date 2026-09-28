// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a navigable token record id from a kanban card payload.
 * Ignores Vue-key fallbacks (row key / array index) that are not record ids.
 */
export function resolveTokenDetailId(payload: Record<string, unknown> | null | undefined): string {
  return String(payload?.Id ?? '').trim();
}
