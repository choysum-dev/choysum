// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Methods that must not run through Onchange/Constraint draft proxies.
 *
 * Class-level Model API (e.g. `OtherModel.Search(...)`) remains allowed; this
 * list only blocks methods reached via draft `this.<name>()`.
 *
 * Aliases cover BaseModel entrypoints, conventional PascalCase services, and
 * common Odoo/ORM persistence names (write/unlink/save/...) so draft `this`
 * cannot smuggle persistence through helper aliases.
 *
 * See `.dev/docs/core/service/record_lifecycle_proxy_wrapper_boundary_plan20260715.md`
 * and `modules/core/service/runtime/RECORD_LIFECYCLE_THIS.md`.
 */

function withCaseVariants(names: string[]): string[] {
  const out = new Set<string>();
  for (const name of names) {
    if (!name) continue;
    out.add(name);
    const camel = name.charAt(0).toLowerCase() + name.slice(1);
    const pascal = name.charAt(0).toUpperCase() + name.slice(1);
    out.add(camel);
    out.add(pascal);
  }
  return [...out];
}

export const DRAFT_FORBIDDEN_PERSISTENCE_METHODS = new Set(
  withCaseVariants([
    // BaseModel instance persistence / refresh.
    'update',
    'delete',
    'reload',
    // Conventional model service / query entrypoints.
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
    // Common Odoo / ORM persistence aliases (may appear as helpers on models).
    'save',
    'upsert',
    'write',
    'unlink',
    'destroy',
    'remove',
    'insert',
    'patch',
    'persist',
  ])
);

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
