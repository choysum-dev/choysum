// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Audience helpers for RoleRecordRule admin Form (PR-C-5).
 *
 * Kind=grant with empty RoleId applies to everyone — warn in the UI.
 * Orthogonal to scope-global (empty Application/Model).
 */
export function isGrantEveryoneWarning(draft: Record<string, any> | null | undefined): boolean {
  if (!draft) return false;
  const kind = String(draft.Kind ?? 'grant')
    .trim()
    .toLowerCase();
  if (kind !== 'grant') return false;
  const role = draft.RoleId;
  if (role == null || role === '') return true;
  if (typeof role === 'object') {
    const id = String((role as { Id?: unknown }).Id ?? '').trim();
    return !id;
  }
  return !String(role).trim();
}
