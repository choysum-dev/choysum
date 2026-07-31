// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { defineModelActions } from '@/core/web/resource';
import { canShowAction, type ActionPredicate } from '@/web/web/components/view/actionVisibility';

/**
 * Resolve UI create action id for NameCreate entry (D6).
 * When `createActionId` prop is passed (including ''), use it; otherwise derive from target model.
 */
export function resolveNameCreateActionId(
  relationQualifiedName: string | undefined | null,
  createActionId?: string
): string | undefined {
  if (createActionId !== undefined) {
    return createActionId;
  }
  const qn = String(relationQualifiedName ?? '').trim();
  if (!qn) return undefined;
  return defineModelActions(qn).create;
}

/** Whether the typeahead Create entry should show (D6). */
export function shouldShowNameCreateEntry(opts: {
  allowCreate: boolean;
  hasKeyword: boolean;
  relationQualifiedName?: string | null;
  createActionId?: string;
  hasAction?: ActionPredicate;
}): boolean {
  if (!opts.allowCreate) return false;
  if (!opts.hasKeyword) return false;
  const actionId = resolveNameCreateActionId(opts.relationQualifiedName, opts.createActionId);
  // Unresolved id (no target / no derived create) → hide. Explicit '' still skips ACL via canShowAction.
  if (actionId === undefined) return false;
  return canShowAction(actionId, opts.hasAction);
}
