// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Resolve a ManyToOne / reference field value to its canonical string ID.
 * Handles object shapes ({Id, id}), primitive strings, and null/undefined.
 */
export function asRefId(value: any): string | null | undefined {
  if (value === undefined) return undefined;
  if (value === null) return null;
  const raw = typeof value === 'object' && value !== null ? (value.Id ?? value.id) : value;
  const id = String(raw ?? '').trim();
  return id ? id : null;
}

/**
 * Resolve the company scope key from a CompanyId reference.
 * Returns '__GLOBAL__' when the company id is empty/null/undefined.
 */
export function normalizeCompanyScopeKey(companyId: any): string {
  const id = asRefId(companyId);
  return id || '__GLOBAL__';
}
