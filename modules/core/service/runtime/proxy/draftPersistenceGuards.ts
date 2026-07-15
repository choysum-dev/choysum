// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Methods that must not run through Onchange/Constraint draft proxies.
 *
 * Class-level Model API (e.g. `OtherModel.Search(...)`) remains allowed; this
 * list only blocks methods reached via draft `this.<name>()`.
 *
 * See `.dev/docs/core/service/record_lifecycle_proxy_wrapper_boundary_plan20260715.md`.
 */
export const DRAFT_FORBIDDEN_PERSISTENCE_METHODS = new Set([
  'update',
  'delete',
  'reload',
  // Common persistence aliases (not currently on BaseModel; block if present on draft `this`).
  'save',
  'Save',
  'upsert',
  'Upsert',
  'Create',
  'CreateMany',
  'Browse',
  'BrowseMany',
  'Search',
  'Update',
  'UpdateById',
  'Delete',
  'DeleteById',
  'Count',
  'Hydrate',
]);

export type DraftPersistenceGuardContext = 'onchange-preview' | 'constraint-draft';

export function createForbiddenPersistenceMethodStub(context: DraftPersistenceGuardContext, methodName: string): () => never {
  return () => {
    if (context === 'onchange-preview') {
      throw new Error(`PREVIEW_METHOD_FORBIDDEN: method "${methodName}" is disabled in onchange preview`);
    }
    throw new Error(`CONSTRAINT_DRAFT_METHOD_FORBIDDEN: method "${methodName}" is disabled in constraint draft`);
  };
}

export function isDraftForbiddenPersistenceMethod(methodName: string): boolean {
  return DRAFT_FORBIDDEN_PERSISTENCE_METHODS.has(methodName);
}
