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
 * Reports whether a view action should be shown for the current actor.
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
