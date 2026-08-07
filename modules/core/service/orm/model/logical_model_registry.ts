// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Process-local registry of @Model short names eligible for auth LogicalModel ACL scope.
 *
 * Platform inject bases (AppSetting / FieldDefault / TranslationTerm / …) call
 * {@link registerLogicalModelName} at module load. Auth write validation reads this set;
 * runtime Method/FieldRule matching still compares short names only (no Ensure flag).
 *
 * See `.dev/docs/auth/logical_model_acl_field_rule_design.md` §5.
 */

const registered = new Set<string>();

/**
 * Register a logical model short name (idempotent). Empty names are ignored.
 */
export function registerLogicalModelName(name: string): void {
  const n = String(name ?? '').trim();
  if (!n) return;
  registered.add(n);
}

/**
 * Whether `name` was registered by a platform inject base (exact short-name match).
 */
export function isRegisteredLogicalModelName(name: string | null | undefined): boolean {
  const n = String(name ?? '').trim();
  return n.length > 0 && registered.has(n);
}

/**
 * Sorted snapshot of registered logical model short names (for admin pickers / tests).
 */
export function listLogicalModelNames(): string[] {
  return Array.from(registered).sort((a, b) => a.localeCompare(b));
}

/**
 * FieldsGet / OSelectionField options for LogicalModelName.
 */
export function listLogicalModelSelection(): Array<{ value: string; label: string }> {
  return listLogicalModelNames().map(name => ({ value: name, label: name }));
}

/**
 * Test-only: clear the registry (does not re-run base-model self-registration).
 */
export function __resetLogicalModelNamesForTest(): void {
  registered.clear();
}
