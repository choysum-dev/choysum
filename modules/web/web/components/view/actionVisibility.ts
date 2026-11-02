// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Standard action slots supported by common web views.
 */
export type ActionIdMap = Partial<Record<'create' | 'edit' | 'delete' | 'copy' | 'refresh', string>>;

/**
 * Predicate used to determine whether an action id is available.
 */
export type ActionPredicate = ((actionId: string | undefined) => boolean) | undefined;

/**
 * ACL gate for a UI action id.
 *
 * - Empty / missing id → true (ungated): used by tool slots such as Refresh, and by
 *   callers that intentionally skip ACL with an explicit empty string.
 * - Non-empty id → consult `hasAction` when provided; missing predicate → allow.
 *
 * Form/List mutating toolbar slots (create/edit/copy/delete) must use
 * {@link canShowDeclaredAction} so omitting an id hides the button (opt-in).
 */
export function canShowAction(actionId: string | undefined, hasAction: ActionPredicate): boolean {
  const id = String(actionId ?? '').trim();
  if (!id) {
    return true;
  }
  if (typeof hasAction !== 'function') {
    return true;
  }
  return !!hasAction(id);
}

/**
 * Opt-in toolbar slot: missing/empty action id → hidden; otherwise ACL via
 * {@link canShowAction}. Prefer this for create/edit/copy/delete on Form/List.
 */
export function canShowDeclaredAction(
  actionId: string | undefined,
  hasAction: ActionPredicate
): boolean {
  const id = String(actionId ?? '').trim();
  if (!id) {
    return false;
  }
  return canShowAction(id, hasAction);
}
