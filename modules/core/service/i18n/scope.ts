// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Scope = path[@location] | manual scope override.
 * See terminology-i18n-design.md §5.3.
 */

const scopeStack: string[] = [];

export function formatScope(path: string, location?: string): string {
  const p = (path || '').trim();
  const loc = (location || '').trim();
  if (!p) {
    return loc || '';
  }
  if (!loc) {
    return p;
  }
  return `${p}@${loc}`;
}

export function withI18nScope<T>(scope: string, fn: () => T): T {
  scopeStack.push(scope);
  try {
    return fn();
  } finally {
    scopeStack.pop();
  }
}

export type ResolveI18nScopeOptions = {
  /** Manual scope override (wins over stack / path). */
  scope?: string;
  /** Source file path used for auto scope. */
  path?: string;
  /** Optional location suffix for auto scope. */
  location?: string;
};

/**
 * Resolve Scope for a `_t` call site.
 * Priority: explicit `{ scope }` → withI18nScope stack → path[@location].
 */
export function resolveI18nScope(opts?: ResolveI18nScopeOptions): string {
  const manual = opts?.scope?.trim();
  if (manual) {
    return manual;
  }
  if (scopeStack.length > 0) {
    return scopeStack[scopeStack.length - 1];
  }
  return formatScope(opts?.path || '', opts?.location);
}

/** Test-only: reset scope stack between cases. */
export function __resetI18nScopeStackForTests(): void {
  scopeStack.length = 0;
}
