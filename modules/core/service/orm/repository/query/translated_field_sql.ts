// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { sql } from 'kysely';
import type { DialectName } from '../repository_dialect';
import { TRANSLATED_BASE_LANG, resolveTranslatedFieldLang } from '../projection/translated_field_codec';
import { repositoryPredicateRef, type RepositoryPredicateBuilder } from './predicate_builder_adapter';

function quoteSqlString(value: string): string {
  return `'${String(value).replace(/'/g, "''")}'`;
}

/**
 * Build a SQL expression that unwraps a translated JSON/JSONB lang map
 * for the current request language (COALESCE current, en_US).
 *
 * Empty string values are preserved (COALESCE only skips NULL), matching
 * data-i18n-design.md §5.1.
 */
export function buildTranslatedFieldUnwrapExpr(
  dialect: DialectName | string,
  eb: RepositoryPredicateBuilder,
  tableQualifiedColumn: string,
  lang?: string
): unknown {
  const resolvedLang = (lang || resolveTranslatedFieldLang()).trim() || TRANSLATED_BASE_LANG;
  const col = repositoryPredicateRef(eb, tableQualifiedColumn);
  const d = String(dialect || 'postgres').toLowerCase() as DialectName | string;
  const langLit = sql.raw(quoteSqlString(resolvedLang));
  const baseLit = sql.raw(quoteSqlString(TRANSLATED_BASE_LANG));

  switch (d) {
    case 'postgres':
    case 'postgresql':
      if (resolvedLang === TRANSLATED_BASE_LANG) {
        return sql`(${col}->>${baseLit})`;
      }
      return sql`COALESCE((${col}->>${langLit}), (${col}->>${baseLit}))`;
    case 'mysql':
    case 'mariadb': {
      const path = (key: string) => sql.raw(quoteSqlString(`$.${key}`));
      const extract = (key: string) => sql`JSON_UNQUOTE(JSON_EXTRACT(${col}, ${path(key)}))`;
      if (resolvedLang === TRANSLATED_BASE_LANG) {
        return extract(TRANSLATED_BASE_LANG);
      }
      return sql`COALESCE(${extract(resolvedLang)}, ${extract(TRANSLATED_BASE_LANG)})`;
    }
    case 'sqlite': {
      const path = (key: string) => sql.raw(quoteSqlString(`$.${key}`));
      const extract = (key: string) => sql`json_extract(${col}, ${path(key)})`;
      if (resolvedLang === TRANSLATED_BASE_LANG) {
        return extract(TRANSLATED_BASE_LANG);
      }
      return sql`COALESCE(${extract(resolvedLang)}, ${extract(TRANSLATED_BASE_LANG)})`;
    }
    case 'mssql':
    case 'sqlserver': {
      const path = (key: string) => sql.raw(quoteSqlString(`$.${key}`));
      const extract = (key: string) => sql`JSON_VALUE(${col}, ${path(key)})`;
      if (resolvedLang === TRANSLATED_BASE_LANG) {
        return extract(TRANSLATED_BASE_LANG);
      }
      return sql`COALESCE(${extract(resolvedLang)}, ${extract(TRANSLATED_BASE_LANG)})`;
    }
    default:
      if (resolvedLang === TRANSLATED_BASE_LANG) {
        return sql`(${col}->>${baseLit})`;
      }
      return sql`COALESCE((${col}->>${langLit}), (${col}->>${baseLit}))`;
  }
}

/** Minimum significant characters for a translated trigram prefilter (data-i18n-design §7.1). */
export const TRANSLATED_TRIGRAM_MIN_CHARS = 3;

function jsonEscapeUnquoted(value: string): string {
  // Match Odoo: json.dumps(value, ensure_ascii=False)[1:-1]
  return JSON.stringify(value).slice(1, -1);
}

function pgWildcardEscape(value: string): string {
  return value.replace(/([_%\\])/g, '\\$1');
}

/**
 * Escape a literal value for translated-field trigram prefilter (exact / `in` with one value).
 * Returns '%' when the value is too short to use the index.
 */
export function valueToTranslatedTrigramPattern(value: string): string {
  const text = String(value ?? '');
  if (text.length < TRANSLATED_TRIGRAM_MIN_CHARS) return '%';
  return `%${pgWildcardEscape(jsonEscapeUnquoted(text))}%`;
}

/**
 * Escape a LIKE/ILIKE pattern for translated-field trigram prefilter.
 * Wildcards (`%`, `_`) stay; text between them is JSON-escaped.
 * Returns '%' when fewer than 3 non-wildcard characters remain (Odoo-compatible).
 */
export function patternToTranslatedTrigramPattern(pattern: string): string {
  const raw = String(pattern ?? '');
  const chunks: string[] = [];
  let textBuf = '';
  let i = 0;
  while (i < raw.length) {
    const ch = raw[i];
    if (ch === '\\' && i + 1 < raw.length) {
      textBuf += raw[i + 1];
      i += 2;
      continue;
    }
    if (ch === '_' || ch === '%') {
      chunks.push(jsonEscapeUnquoted(textBuf) + ch);
      textBuf = '';
      i += 1;
      continue;
    }
    textBuf += ch;
    i += 1;
  }
  chunks.push(jsonEscapeUnquoted(textBuf));
  const escapedPattern = chunks.join('');

  let significantLen = 0;
  for (let j = 0; j < escapedPattern.length; j++) {
    const c = escapedPattern[j];
    if (c === '\\' && j + 1 < escapedPattern.length) {
      significantLen += 1;
      j += 1;
      continue;
    }
    if (c === '_' || c === '%') continue;
    significantLen += 1;
  }
  if (significantLen < TRANSLATED_TRIGRAM_MIN_CHARS) return '%';
  return escapedPattern;
}

export function fieldHasTranslatedTrigramIndex(fieldMeta: { translate?: boolean; column?: { index?: unknown } } | undefined): boolean {
  if (!fieldMeta?.translate) return false;
  return String(fieldMeta.column?.index || '').trim().toLowerCase() === 'trigram';
}

/**
 * Full-language text expression used by the PG trigram GIN index.
 */
export function buildTranslatedTrigramPrefilterLhs(
  eb: RepositoryPredicateBuilder,
  tableQualifiedColumn: string
): unknown {
  const col = repositoryPredicateRef(eb, tableQualifiedColumn);
  return sql`(jsonb_path_query_array(${col}, '$.*')::text)`;
}

export function resolveTranslatedTrigramPrefilterPattern(op: string, rhs: unknown): string | null {
  const lowerOp = String(op || '').toLowerCase();
  if (lowerOp === 'in') {
    if (!Array.isArray(rhs) || rhs.length !== 1) return null;
    const only = rhs[0];
    if (typeof only !== 'string') return null;
    const pattern = valueToTranslatedTrigramPattern(only);
    return pattern === '%' ? null : pattern;
  }
  if (lowerOp === 'like' || lowerOp === 'ilike' || lowerOp === '=like' || lowerOp === '=ilike') {
    const text = typeof rhs === 'string' ? rhs : String(rhs ?? '');
    const pattern = patternToTranslatedTrigramPattern(text);
    return pattern === '%' ? null : pattern;
  }
  if (lowerOp === '=' || lowerOp === '==') {
    if (typeof rhs !== 'string') return null;
    const pattern = valueToTranslatedTrigramPattern(rhs);
    return pattern === '%' ? null : pattern;
  }
  return null;
}
