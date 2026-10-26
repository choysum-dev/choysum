// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Restore keyboard focus to the pre-dialog trigger when it is still in the
 * document. Detached nodes are skipped so focus is not left on a silent no-op.
 */
export function restoreDialogFocus(
  el: { focus?: () => void } | null | undefined,
  doc: Pick<Document, 'contains'> = document,
): boolean {
  if (!el || typeof el.focus !== 'function') return false;
  if (!doc.contains(el as Node)) return false;
  el.focus();
  return true;
}
