// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { canShowAction, type ActionPredicate } from '@/web/web/components/view/actionVisibility';

/**
 * Derive the conventional create UI action id for a qualified model name.
 * Mirrors defineModelActions(`${app}.${Model}`).create without calling it
 * (UI_DECL_004 forbids non-literal defineModelActions args).
 */
export function deriveModelCreateActionId(relationQualifiedName: string | undefined | null): string | undefined {
  const qn = String(relationQualifiedName ?? '').trim();
  if (!qn) return undefined;
  const [appRaw, modelRaw] = qn.split('.');
  const app = String(appRaw || '').trim();
  const modelName = String(modelRaw || '').trim();
  if (!app || !modelName) return undefined;
  return `${app}.action.${toSnake(modelName)}_create`;
}

function toSnake(input: string): string {
  const value = String(input || '').trim();
  if (!value) return '';
  return value
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .replace(/[-\s]+/g, '_')
    .toLowerCase();
}

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
  return deriveModelCreateActionId(relationQualifiedName);
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
  // Unresolved id (no target / invalid qn) → hide. Explicit '' still skips ACL via canShowAction.
  if (actionId === undefined) return false;
  return canShowAction(actionId, opts.hasAction);
}
