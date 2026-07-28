// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { normalizeOptionalString, normalizeStringArray } from '../utils/normalization';
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
  if (Array.isArray(value)) {
    const ids = normalizeStringArray(value).sort();
    return ids.length ? ids : undefined;
  }
  if (typeof value === 'string') {
    const ids = normalizeStringArray(value.split(',')).sort();
    return ids.length ? ids : undefined;
  }
  return undefined;
}

/**
 * Normalize a loose value into a condition-envelope shape.
 */
export function normalizeConditionEnvelope(value: unknown): ConditionEnvelope {
  const record = asPlainRecord(value);
  if (!record) return { kind: 'false', reason: 'invalid_record_rule_envelope' };

  const kind = normalizeOptionalString(record.kind);
  const reason = normalizeOptionalString(record.reason);
  const hitRuleIds = normalizeHitRuleIds(record.hitRuleIds);
  if (kind === 'true') {
    return hitRuleIds ? { kind: 'true', reason, hitRuleIds } : { kind: 'true', reason };
  }
  if (kind === 'false') {
    return hitRuleIds ? { kind: 'false', reason, hitRuleIds } : { kind: 'false', reason };
  }
  if (kind === 'expr') {
    const exprIsArray = Array.isArray(record.expr);
    const exprRecord = asPlainRecord(record.expr);
    if (exprIsArray || exprRecord) {
      return hitRuleIds
        ? {
            kind: 'expr',
            expr: record.expr as ConditionExpr,
            reason,
            hitRuleIds,
          }
        : {
            kind: 'expr',
            expr: record.expr as ConditionExpr,
            reason,
          };
    }
  }

  return { kind: 'false', reason: 'invalid_record_rule_envelope' };
}

export type FieldRuleSpec = {
  denyReadFields: string[];
  denyWriteFields: string[];
  reason?: string;
  hitRuleIds?: string[];
};

/**
 * Normalize a loose value into a field-rule spec shape.
 */
export function normalizeFieldRuleSpec(value: unknown): FieldRuleSpec {
  const record = asPlainRecord(value);
  if (!record) {
    return { denyReadFields: [], denyWriteFields: [] };
  }

  const hitRuleIds = normalizeHitRuleIds(record.hitRuleIds);
  return {
    denyReadFields: normalizeStringArray(record.denyReadFields),
    denyWriteFields: normalizeStringArray(record.denyWriteFields),
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
