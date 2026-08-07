// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

/**
 * Auth-side helpers for LogicalModel ACL scope.
 *
 * Short-name eligibility comes from core platform-inject bases via
 * {@link registerLogicalModelName} (not an auth constant list).
 * See `.dev/docs/auth/logical_model_acl_field_rule_design.md` §5.
 */

// Ensure platform store bases have self-registered before write validation.
import '@/core/service/orm/model/app_setting_base_model';
import '@/core/service/orm/model/field_default_base_model';
import '@/core/service/orm/model/translation_term_base_model';
import {
  isRegisteredLogicalModelName,
  listLogicalModelNames,
  listLogicalModelSelection,
} from '@/core/service/orm/model/logical_model_registry';

export { isRegisteredLogicalModelName, listLogicalModelNames, listLogicalModelSelection };

/**
 * Normalize a logical model short name for write validation.
 * Empty → null. Non-empty unregistered → throws.
 */
export function normalizeLogicalModelName(raw: unknown): string | null {
  if (raw == null) return null;
  const n = String(raw).trim();
  if (!n) return null;
  if (!isRegisteredLogicalModelName(n)) {
    throw new Error(`invalid LogicalModelName: ${n} is not a registered logical model`);
  }
  return n;
}

/**
 * Canonicalize one RPC short name: trim, drop empties, PascalCase first letter.
 * Comparison at eval time is case-insensitive (toLowerCase).
 */
export function canonicalizeLogicalMethodName(raw: unknown): string {
  const t = String(raw ?? '').trim();
  if (!t) return '';
  return t.charAt(0).toUpperCase() + t.slice(1);
}

/**
 * Normalize LogicalMethods JSON payload to string[] | null.
 * null / [] / missing → null (meaning "all methods" at eval time).
 */
export function normalizeLogicalMethods(raw: unknown): string[] | null {
  if (raw == null) return null;
  let arr: unknown[] | null = null;
  if (Array.isArray(raw)) {
    arr = raw;
  } else if (typeof raw === 'string') {
    const s = raw.trim();
    if (!s) return null;
    try {
      const parsed = JSON.parse(s);
      if (Array.isArray(parsed)) arr = parsed;
      else throw new Error('not an array');
    } catch {
      throw new Error('invalid LogicalMethods: must be a JSON string array');
    }
  } else {
    throw new Error('invalid LogicalMethods: must be a string array');
  }

  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of arr) {
    const name = canonicalizeLogicalMethodName(item);
    if (!name) continue;
    const key = name.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(name);
  }
  return out.length === 0 ? null : out;
}

/**
 * Whether a LogicalMethods whitelist (null/empty = all) covers `methodName`.
 */
export function logicalMethodsAllow(methods: unknown, methodName: string): boolean {
  const want = String(methodName || '')
    .trim()
    .toLowerCase();
  if (!want) return false;
  const list = normalizeLogicalMethods(methods);
  if (list == null) return true;
  return list.some(m => m.toLowerCase() === want);
}
