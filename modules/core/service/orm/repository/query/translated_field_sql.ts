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
