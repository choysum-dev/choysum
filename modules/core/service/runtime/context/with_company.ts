// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { withContext } from './scope';

/**
 * Temporary company-view target for {@link withCompany}.
 *
 * - string: active company; enabled defaults to `[active]`
 * - object: active required; omit enabled ⇒ `[active]`
 *
 * Does **not** clamp against `identity.allowedCompanyIds` (author responsibility).
 */
export type WithCompanyTarget = string | { activeCompanyId: string; enabledCompanyIds?: string[] };

export type WithCompanyPatch = { activeCompanyId: string; enabledCompanyIds: string[] };

function trimNonEmpty(value: unknown): string | undefined {
  const normalized = typeof value === 'string' ? value.trim() : String(value ?? '').trim();
  return normalized || undefined;
}

/** Order-preserving unique non-empty strings. */
function dedupePreserveOrder(values: unknown[]): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of values) {
    const id = trimNonEmpty(item);
    if (!id || seen.has(id)) continue;
    seen.add(id);
    out.push(id);
  }
  return out;
}

/**
 * Normalizes a {@link WithCompanyTarget} into a business-ctx patch for `withContext`.
 * Blank active company Id throws (fail-closed).
 */
export function normalizeWithCompanyPatch(company: WithCompanyTarget): WithCompanyPatch {
  if (typeof company === 'string' || company == null || typeof company !== 'object') {
    const active = trimNonEmpty(company);
    if (!active) {
      throw new Error('withCompany requires a non-empty activeCompanyId');
    }
    return { activeCompanyId: active, enabledCompanyIds: [active] };
  }

  const active = trimNonEmpty(company.activeCompanyId);
  if (!active) {
    throw new Error('withCompany requires a non-empty activeCompanyId');
  }

  if (company.enabledCompanyIds === undefined) {
    return { activeCompanyId: active, enabledCompanyIds: [active] };
  }

  const enabled = dedupePreserveOrder(Array.isArray(company.enabledCompanyIds) ? company.enabledCompanyIds : [company.enabledCompanyIds]);
  return { activeCompanyId: active, enabledCompanyIds: enabled };
}

/**
 * Runs `fn` with a temporary company view (`activeCompanyId` / `enabledCompanyIds`).
 *
 * Thin wrapper over {@link withContext} with `{ merge: true }`. Does not change
 * authz userId, does not bypass RecordRule/FieldRule, and does **not** validate
 * or clamp against `allowedCompanyIds` — session trust remains at token /
 * SwitchCompanyScope. Prefer subsets of the caller's allowed companies.
 */
export function withCompany<R>(company: WithCompanyTarget, fn: () => R): R {
  return withContext(normalizeWithCompanyPatch(company), fn, { merge: true });
}
