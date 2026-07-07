// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { QueryCondition } from '@/core/service/api/query';
import type { ConditionGroup } from '../../types';
import { isLikeOperator, normalizeLikeValue } from './like';
import { normalizeFields } from '@/core/service/utils/normalization';

/**
 * Options used when deriving keyword fields from metadata.
 */
export type ResolveKeywordFieldsOptions = {
  fieldsMeta?: Record<string, any>;
  fallbackTextTypes?: string[];
  includeIdInFallback?: boolean;
  mapDisplayNameToName?: boolean;
  fallbackWhenFilteredEmpty?: boolean;
};

/**
 * Options used when building a keyword condition.
 */
export type BuildKeywordConditionOptions = ResolveKeywordFieldsOptions & {
  operator?: 'like' | 'ilike';
  normalizeLike?: boolean;
};

/**
 * Converts a condition group into a backend query condition recursively.
 */
function groupToExpr(node: ConditionGroup): QueryCondition<any> | null {
  if (!node) return null;
  const logic = node.logic;
  const children = Array.isArray((node as any).children) ? (node as any).children : [];
  const parts: QueryCondition<any>[] = [];
  for (const ch of children) {
    if ((ch as any).children) {
      const sub = groupToExpr(ch as ConditionGroup);
      if (sub) parts.push(sub);
    } else {
      const c = ch as any;
      if (!c.field || !c.operator) continue;
      let v = c.value;
      if (isLikeOperator(c.operator)) v = normalizeLikeValue(v);
      parts.push([c.field, c.operator, v] as unknown as QueryCondition<any>);
    }
  }
  if (parts.length === 0) return null;
  if (parts.length === 1) return parts[0];
  return logic === 'Or' ? ({ Or: parts } as any) : ({ And: parts } as any);
}

/**
 * Derives keyword-searchable fields from metadata when callers do not provide them.
 */
export function deriveKeywordFieldsFromMeta(fieldsMeta?: Record<string, any>, options?: { textTypes?: string[]; includeId?: boolean }): string[] {
  try {
    const md = fieldsMeta || {};
    const textTypes = Array.isArray(options?.textTypes) && options!.textTypes!.length > 0 ? options!.textTypes! : ['char', 'varchar'];
    const allowed = new Set(textTypes.map(t => String(t || '').toLowerCase()));
    const derived = Object.keys(md).filter(k => {
      const t = String(md[k]?.type || '').toLowerCase();
      return allowed.has(t);
    });
    if (options?.includeId && Object.prototype.hasOwnProperty.call(md, 'Id')) {
      return Array.from(new Set(['Id', ...derived]));
    }
    return derived;
  } catch {
    return [];
  }
}

/**
 * Resolves keyword fields against metadata and fallback rules.
 */
export function resolveKeywordFieldsByMeta(keywordFields: string[] | undefined, options?: ResolveKeywordFieldsOptions): string[] {
  const preferred = normalizeFields(keywordFields);
  const fieldsMeta = options?.fieldsMeta;

  if (!fieldsMeta || Object.keys(fieldsMeta).length === 0) {
    return preferred;
  }

  const keys = new Set(Object.keys(fieldsMeta));
  const mapDisplayNameToName = options?.mapDisplayNameToName !== false;
  const mapped: string[] = [];

  for (const field of preferred) {
    if (keys.has(field)) {
      mapped.push(field);
      continue;
    }
    if (field === 'DisplayName' && mapDisplayNameToName && keys.has('Name')) {
      mapped.push('Name');
    }
  }

  const uniqMapped = Array.from(new Set(mapped));
  if (uniqMapped.length > 0) return uniqMapped;
  if (!options?.fallbackWhenFilteredEmpty) return [];

  return deriveKeywordFieldsFromMeta(fieldsMeta, {
    textTypes: options?.fallbackTextTypes,
    includeId: options?.includeIdInFallback,
  });
}

/**
 * Builds a keyword condition over the resolved keyword fields.
 */
export function buildKeywordCondition(
  keyword: string | undefined,
  keywordFields: string[] | undefined,
  options?: BuildKeywordConditionOptions
): QueryCondition<any> | null {
  const kw = (keyword || '').trim();
  if (!kw) return null;

  const resolvedFields = resolveKeywordFieldsByMeta(keywordFields, options);
  if (!resolvedFields || resolvedFields.length === 0) return null;

  const operator = options?.operator ?? 'like';
  const normalize = options?.normalizeLike ?? operator === 'like';
  const value = normalize ? normalizeLikeValue(kw) : kw;

  const parts: QueryCondition<any>[] = resolvedFields.map(f => [f, operator, value] as unknown as QueryCondition<any>);
  if (parts.length === 1) return parts[0];
  return { Or: parts } as any;
}

/**
 * Combines UI filters and keyword search into a backend query condition.
 */
export function filtersToQuery(
  filters: ConditionGroup[] | undefined,
  keyword?: string,
  keywordFields?: string[],
  fieldsMeta?: Record<string, any>
): QueryCondition<any> | [] {
  const fArr = Array.isArray(filters) ? filters : [];
  const parts: QueryCondition<any>[] = [];
  for (const g of fArr) {
    const expr = groupToExpr(g);
    if (expr) parts.push(expr);
  }
  const hasExplicitKeywordFields = Array.isArray(keywordFields) && keywordFields.length > 0;
  const kwExpr = buildKeywordCondition(keyword, keywordFields, {
    fieldsMeta,
    operator: 'like',
    normalizeLike: true,
    fallbackTextTypes: ['char', 'varchar'],
    includeIdInFallback: false,
    mapDisplayNameToName: true,
    fallbackWhenFilteredEmpty: !hasExplicitKeywordFields,
  });
  if (kwExpr) parts.push(kwExpr);
  if (parts.length === 0) return [];
  if (parts.length === 1) return parts[0];
  return { And: parts } as any;
}
