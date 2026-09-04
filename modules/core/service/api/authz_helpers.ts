// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString } from '../utils/normalization';
import type { ConditionExpr, ConditionEnvelope } from './authz';

function asPlainRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return null;
  const proto = Object.getPrototypeOf(value);
  if (proto !== Object.prototype && proto !== null) return null;
  return value as Record<string, unknown>;
}

/**
 * Normalize hit rule Ids from array or comma-separated string payloads.
 */
export function normalizeHitRuleIds(value: unknown): string[] | undefined {
  const toIds = (items: unknown[]): string[] =>
    Array.from(new Set(items.map(item => String(item ?? '').trim()).filter(Boolean))).sort();

  if (Array.isArray(value)) {
    const ids = toIds(value);
    return ids.length ? ids : undefined;
  }
  if (typeof value === 'string') {
    const ids = toIds(value.split(','));
    return ids.length ? ids : undefined;
  }
  return undefined;
}

/**
 * True when value matches the repository condition-tree shape (tuple / And / Or).
 */
function isConditionExprShape(value: unknown, depth = 0): value is ConditionExpr {
  if (depth > 32) return false;

  if (Array.isArray(value)) {
    return value.length === 3 && typeof value[0] === 'string' && typeof value[1] === 'string';
  }

  const record = asPlainRecord(value);
  if (!record) return false;

  const keys = Object.keys(record);
  if (keys.length !== 1) return false;

  const key = keys[0];
  if (key !== 'And' && key !== 'Or') return false;

  const children = record[key];
  if (!Array.isArray(children) || children.length === 0) return false;
  return children.every(child => isConditionExprShape(child, depth + 1));
}

/**
 * Parse a loose value into a typed condition envelope; throws when shape is invalid.
 */
export function parseConditionEnvelopeFromUnknown(value: unknown): ConditionEnvelope {
  const record = asPlainRecord(value);
  if (!record) throw new Error('invalid_record_rule_envelope');

  const kind = normalizeOptionalString(record.kind);
  const reason = normalizeOptionalString(record.reason);
  const hitRuleIds = normalizeHitRuleIds(record.hitRuleIds);
  const diagnostics = { reason, ...(hitRuleIds ? { hitRuleIds } : {}) };
  if (kind === 'true') return { kind: 'true', ...diagnostics };
  if (kind === 'false') return { kind: 'false', ...diagnostics };
  if (kind === 'expr') {
    if (!isConditionExprShape(record.expr)) {
      throw new Error('invalid_record_rule_envelope');
    }
    return { kind: 'expr', expr: record.expr, ...diagnostics };
  }

  throw new Error('invalid_record_rule_envelope');
}

export type FieldRuleSpec = {
  denyReadFields: string[];
  denyWriteFields: string[];
  reason?: string;
  hitRuleIds?: string[];
};

function parseDenyFieldList(value: unknown): string[] {
  if (value === undefined) return [];
  if (!Array.isArray(value)) throw new Error('invalid_field_rule_spec');

  const out: string[] = [];
  const seen = new Set<string>();
  for (const item of value) {
    if (typeof item !== 'string') throw new Error('invalid_field_rule_spec');
    const field = item.trim();
    if (!field) throw new Error('invalid_field_rule_spec');
    if (seen.has(field)) continue;
    seen.add(field);
    out.push(field);
  }
  return out;
}

/**
 * Format parse/authz failure detail for permission-denied metadata.
 */
export function formatAuthzParseFailureDetail(error: unknown): string {
  if (error instanceof Error) return error.message;
  return String(error);
}

/**
 * Parse a loose value into a typed field-rule spec; throws when shape is invalid.
 *
 * Missing deny lists default to empty arrays. Present non-array deny lists or
 * non-string / blank deny-list elements throw (do not wash to allow-all).
 */
export function parseFieldRuleSpecFromUnknown(value: unknown): FieldRuleSpec {
  const record = asPlainRecord(value);
  if (!record) throw new Error('invalid_field_rule_spec');

  const hitRuleIds = normalizeHitRuleIds(record.hitRuleIds);
  return {
    denyReadFields: parseDenyFieldList(record.denyReadFields),
    denyWriteFields: parseDenyFieldList(record.denyWriteFields),
    reason: normalizeOptionalString(record.reason),
    ...(hitRuleIds ? { hitRuleIds } : {}),
  };
}

export type ConditionTokenValues = {
  userId?: string;
  companyId?: string;
  companyIds?: string[];
  strictUnknownToken?: boolean;
};

/**
 * Replace well-known condition tokens recursively in a condition expression.
 */
export function replaceConditionExprTokens(expr: ConditionExpr, values: ConditionTokenValues): ConditionExpr {
  const replace = (value: unknown): unknown => {
    if (value === null || value === undefined) return value;

    if (typeof value === 'string') {
      const token = value.trim();
      if (token === '$userId') {
        if (!values.userId) throw new Error('missing token value: $userId');
        return values.userId;
      }
      if (token === '$companyId') {
        if (!values.companyId) throw new Error('missing token value: $companyId');
        return values.companyId;
      }
      if (token === '$companyIds') {
        if (!values.companyIds || values.companyIds.length === 0) throw new Error('missing token value: $companyIds');
        return values.companyIds;
      }

      if (values.strictUnknownToken && /^\$[A-Za-z_][A-Za-z0-9_]*$/.test(token)) {
        throw new Error(`unknown condition token: ${token}`);
      }
      return value;
    }

    if (Array.isArray(value)) return value.map(item => replace(item));

    const record = asPlainRecord(value);
    if (!record) return value;

    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(record)) {
      out[k] = replace(v);
    }
    return out;
  };

  return replace(expr) as ConditionExpr;
}
