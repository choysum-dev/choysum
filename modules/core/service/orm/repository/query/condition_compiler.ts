// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getStringHelpers } from './string_helpers';
import { FieldMetadata, ManyToOneMetadata, MetadataStorage, ModelCtor, ModelMetadata, SelectSubqueryBuilder } from '../../metadata';
import type BaseModel from '../../model/model';
import { BaseQueryCondition } from '../types';
import type { DialectName } from '../repository_dialect';
import { asBigdecimal } from '@/core/utils/decimal';
import { buildContainsExpression } from './json_contains';
import { sql } from 'kysely';
import type { DbLike } from './select_context';
import {
  createRepositoryPredicateSelectCtx,
  repositoryPredicateAnd,
  repositoryPredicateCall,
  repositoryPredicateOr,
  repositoryPredicateRef,
  type RepositoryPredicate,
  type RepositoryPredicateBuilder,
} from './predicate_builder_adapter';
import { hasRepositorySqlComputeExpression, resolveRepositorySqlComputeExpression } from './sql_compute_expression';
import { rewriteSearchCondition } from '../../../runtime/compute/search_rewrite';
import {
  buildTranslatedFieldUnwrapExpr,
  buildTranslatedTrigramPrefilterLhs,
  fieldHasTranslatedTrigramIndex,
  resolveTranslatedTrigramPrefilterPattern,
} from './translated_field_sql';
import { buildCompanyDependentFieldUnwrapExpr } from './company_dependent_field_sql';

function supportsContainsFieldType(fieldMeta: FieldMetadata | undefined): boolean {
  const fieldType = fieldMeta?.type;
  return fieldType === 'jsonobject' || fieldType === 'ManyToManyRef';
}

/** Prefer explicit column.name when present (ManyToOne FK remap); else logical field name. */
function resolveStoredColumnName(fieldMeta: FieldMetadata, fieldName: string): string {
  const column = (fieldMeta as { column?: unknown }).column;
  if (column == null) return fieldName;
  if (typeof column !== 'object') return fieldName;
  if (!('name' in column)) return fieldName;
  const name = (column as { name?: unknown }).name;
  if (typeof name !== 'string') return fieldName;
  const trimmed = name.trim();
  if (!trimmed) return fieldName;
  return trimmed;
}

function isPostgresDialect(dialect: string): boolean {
  const d = String(dialect || '').toLowerCase();
  return d === 'postgres' || d === 'postgresql';
}

/**
 * Optionally AND a full-language trigram prefilter with the unwrap predicate (S3-2 / §7.1).
 * Skips when not PG, field lacks index:'trigram', or the pattern is too short.
 */
function maybeAndTranslatedTrigramPrefilter(
  dialect: string,
  eb: RepositoryPredicateBuilder,
  selfTable: string,
  fieldName: string,
  fieldMeta: FieldMetadata | undefined,
  op: unknown,
  rhs: unknown,
  exactPredicate: RepositoryPredicate
): RepositoryPredicate {
  if (!isPostgresDialect(dialect) || !fieldHasTranslatedTrigramIndex(fieldMeta)) {
    return exactPredicate;
  }
  const prefilterPattern = resolveTranslatedTrigramPrefilterPattern(String(op || ''), rhs);
  if (!prefilterPattern) {
    return exactPredicate;
  }
  const lowerOp = String(op || '').toLowerCase();
  const prefilterOp = lowerOp === 'in' || lowerOp === '=' || lowerOp === '==' ? 'like' : String(op || '');
  const lhs = buildTranslatedTrigramPrefilterLhs(eb, `${selfTable}.${fieldName}`);
  const prefilter = repositoryPredicateCall(eb, lhs, prefilterOp, prefilterPattern);
  return repositoryPredicateAnd(eb, [prefilter, exactPredicate]);
}

export function convertCondition(
  db: DbLike,
  getDialect: () => string,
  meta: ModelMetadata,
  eb: RepositoryPredicateBuilder,
  condition: BaseQueryCondition,
  selfTable?: string
): RepositoryPredicate {
  const modelLabel = String(meta.fullModelName || meta.modelName || meta.className || meta.type?.name || 'Unknown');
  const createSelectCtx = (table: string) => createRepositoryPredicateSelectCtx(db, getDialect, eb, table, meta);

  const wrapIfDecimal = (fieldName: string | undefined, op: unknown, rhs: unknown) => {
    if (!fieldName) return rhs;
    const fieldMeta = meta.fields.get(fieldName);
    if (fieldMeta?.type !== 'decimal' && fieldMeta?.type !== 'monetary') return rhs;

    const lowerOp = String(op || '').toLowerCase();
    if (lowerOp === 'in' || lowerOp === 'not in') {
      return Array.isArray(rhs) ? rhs.map(value => asBigdecimal(value)) : rhs;
    }
    if (lowerOp === 'between' || lowerOp === 'not between') {
      return Array.isArray(rhs) && rhs.length === 2 ? [asBigdecimal(rhs[0]), asBigdecimal(rhs[1])] : rhs;
    }
    return asBigdecimal(rhs);
  };

  const resolveLeafMetaForPath = (path: string) => {
    const parts = path.split('.').filter(Boolean);
    if (!parts.length) return undefined as FieldMetadata | undefined;

    let currentMeta = meta;
    let leafMeta: FieldMetadata | undefined;
    for (let index = 0; index < parts.length; index++) {
      const segment = parts[index];
      const fieldMeta = currentMeta.fields.get(segment) as FieldMetadata | undefined;
      if (!fieldMeta) return undefined;
      leafMeta = fieldMeta;
      const isLeaf = index === parts.length - 1;
      if (isLeaf) break;
      if (fieldMeta.type !== 'ManyToOne' || !fieldMeta.relation) return undefined;
      const ctor = (fieldMeta.relation as ManyToOneMetadata<BaseModel>).targetModel?.();
      if (!ctor) return undefined;
      currentMeta = MetadataStorage.instance.getModelMetadata(ctor);
    }

    return leafMeta;
  };

  const wrapByMeta = (fieldMeta: FieldMetadata | undefined, op: unknown, rhs: unknown) => {
    if (!fieldMeta || (fieldMeta.type !== 'decimal' && fieldMeta.type !== 'monetary')) return rhs;
    const lowerOp = String(op || '').toLowerCase();
    if (lowerOp === 'in' || lowerOp === 'not in') {
      return Array.isArray(rhs) ? rhs.map(value => asBigdecimal(value)) : rhs;
    }
    if (lowerOp === 'between' || lowerOp === 'not between') {
      return Array.isArray(rhs) && rhs.length === 2 ? [asBigdecimal(rhs[0]), asBigdecimal(rhs[1])] : rhs;
    }
    return asBigdecimal(rhs);
  };

  const asPredicate = (current: BaseQueryCondition): RepositoryPredicate => {
    if (Array.isArray(current)) {
      if (current.length === 0) return repositoryPredicateAnd(eb, []);
      if (current.length !== 3) {
        throw new Error(`invalid condition tuple length: ${current.length}`);
      }

      const [lhs, op, rhs] = current as [string, unknown, unknown];
      const fieldName = typeof lhs === 'string' ? lhs : undefined;

      let effectiveOp = op;
      const effectiveRhs = rhs;
      const rawLowerOp = String(op || '').toLowerCase();
      if (effectiveRhs == null) {
        if (rawLowerOp === '=' || rawLowerOp === '==') {
          effectiveOp = 'is';
        } else if (rawLowerOp === '!=' || rawLowerOp === '<>') {
          effectiveOp = 'is not';
        }
      }
      const lowerOp = String(effectiveOp || '').toLowerCase();

      if (typeof fieldName === 'string' && !fieldName.includes('.')) {
        const rewritten = rewriteSearchCondition(meta, fieldName, effectiveOp, effectiveRhs, String(getDialect() || 'postgres') as DialectName, 'query');
        if (rewritten) {
          if (rewritten.kind === 'domain') {
            return asPredicate(rewritten.domain);
          }
          return rewritten.sql as RepositoryPredicate;
        }
      }

      if (lowerOp === 'ilike' || lowerOp === 'not ilike') {
        const dialect = (getDialect() as DialectName) || 'postgres';
        const { lower } = getStringHelpers(dialect, eb);

        let lhsExpr: unknown;
        if (typeof fieldName === 'string' && selfTable) {
          if (fieldName.includes('.')) {
            const modelCtor = meta.type as unknown as ModelCtor<BaseModel>;
            lhsExpr = createSelectCtx(selfTable).field(modelCtor, fieldName);
          } else {
            if (hasRepositorySqlComputeExpression(meta, fieldName)) {
              const resolved = resolveRepositorySqlComputeExpression(meta, fieldName, createSelectCtx(selfTable));
              if (resolved === undefined) {
                throw new Error(`field sql compute handler is missing: ${modelLabel}.${fieldName}`);
              }
              lhsExpr = resolved;
            } else if (meta.fields.get(fieldName)?.translate) {
              lhsExpr = buildTranslatedFieldUnwrapExpr(dialect, eb, `${selfTable}.${fieldName}`);
            } else {
              const leafMeta = meta.fields.get(fieldName);
              if (leafMeta?.companyDependent) {
                const col = resolveStoredColumnName(leafMeta, fieldName);
                lhsExpr = buildCompanyDependentFieldUnwrapExpr(dialect, eb, `${selfTable}.${col}`);
              } else {
                lhsExpr = repositoryPredicateRef(eb, `${selfTable}.${fieldName}`);
              }
            }
          }
        } else {
          lhsExpr = repositoryPredicateRef(eb, lhs);
        }

        const pattern = typeof effectiveRhs === 'string' ? effectiveRhs : String(effectiveRhs ?? '');
        let exact: RepositoryPredicate;
        if (String(dialect).toLowerCase() === 'postgres') {
          exact = repositoryPredicateCall(eb, lhsExpr, effectiveOp, pattern);
        } else {
          const mapped = lowerOp === 'not ilike' ? 'not like' : 'like';
          exact = repositoryPredicateCall(eb, lower(lhsExpr as Parameters<typeof lower>[0]), mapped, pattern.toLowerCase());
        }

        // Trigram prefilter only for positive ilike (not `not ilike`), matching Odoo.
        if (
          lowerOp === 'ilike' &&
          typeof fieldName === 'string' &&
          selfTable &&
          !fieldName.includes('.')
        ) {
          return maybeAndTranslatedTrigramPrefilter(
            String(dialect),
            eb,
            selfTable,
            fieldName,
            meta.fields.get(fieldName),
            effectiveOp,
            pattern,
            exact
          );
        }
        return exact;
      }

      if (lowerOp === 'contains') {
        const dialect = (getDialect() as DialectName) || 'postgres';

        if (typeof fieldName === 'string') {
          const fieldMeta = meta.fields.get(fieldName);
          if (fieldMeta && !supportsContainsFieldType(fieldMeta) && !fieldMeta.companyDependent) {
            console.warn(
              `[Query] contains is recommended only for JSON container fields (currently jsonobject, ManyToManyRef, or expressions selectable as JSON), but field "${fieldName}" has type "${fieldMeta.type}"`
            );
          }
        }

        let lhsExpr: unknown;
        if (typeof fieldName === 'string' && selfTable) {
          if (fieldName.includes('.')) {
            const modelCtor = meta.type as unknown as ModelCtor<BaseModel>;
            lhsExpr = createSelectCtx(selfTable).field(modelCtor, fieldName);
          } else {
            if (hasRepositorySqlComputeExpression(meta, fieldName)) {
              const resolved = resolveRepositorySqlComputeExpression(meta, fieldName, createSelectCtx(selfTable));
              if (resolved === undefined) {
                throw new Error(`field sql compute handler is missing: ${modelLabel}.${fieldName}`);
              }
              lhsExpr = resolved;
            } else {
              const leafMeta = meta.fields.get(fieldName);
              if (leafMeta?.companyDependent) {
                // Unwrap active-company scalar so contains does not match the whole company map blob.
                const col = resolveStoredColumnName(leafMeta, fieldName);
                lhsExpr = buildCompanyDependentFieldUnwrapExpr(dialect, eb, `${selfTable}.${col}`);
              } else {
                lhsExpr = repositoryPredicateRef(eb, `${selfTable}.${fieldName}`);
              }
            }
          }
        } else {
          lhsExpr = repositoryPredicateRef(eb, lhs);
        }

        return buildContainsExpression(dialect, eb, lhsExpr, effectiveRhs, selfTable, fieldName) as RepositoryPredicate;
      }

      if ((lowerOp === 'child_of' || lowerOp === 'parent_of') && fieldName === 'Id') {
        if (!selfTable) throw new Error(`${lowerOp} requires selfTable`);
        if (!meta.parentField) throw new Error(`Model ${meta.modelName || meta.className} does not configure parentField and cannot use ${lowerOp}`);

        const table = meta.tableName();
        const idField = meta.fields.get('Id');
        const idColSelf = idField ? resolveStoredColumnName(idField, 'Id') : 'Id';
        const parentPathField = meta.fields.get('ParentPath');
        const parentPathCol = parentPathField ? resolveStoredColumnName(parentPathField, 'ParentPath') : 'ParentPath';
        const dialect = String(getDialect() || 'postgres').toLowerCase();

        const sourcePathSubquery = db
          .selectFrom(`${table} as s`)
          .select(`s.${parentPathCol}`)
          .where(`s.${idColSelf}`, '=', effectiveRhs)
          .where(`s.deleted_at`, 'is', null)
          .limit(1);

        let subquery: SelectSubqueryBuilder;
        if (lowerOp === 'child_of') {
          const patternExpr = dialect === 'mysql' ? sql`concat((${sourcePathSubquery}), '%')` : sql`(${sourcePathSubquery}) || '%'`;
          subquery = db.selectFrom(`${table} as t`).select(`t.${idColSelf}`).where(`t.${parentPathCol}`, 'like', patternExpr).where(`t.deleted_at`, 'is', null);
        } else {
          const patternExpr =
            dialect === 'mysql'
              ? sql`concat(${repositoryPredicateRef(eb, `t.${parentPathCol}`)}, '%')`
              : sql`${repositoryPredicateRef(eb, `t.${parentPathCol}`)} || '%'`;
          subquery = db
            .selectFrom(`${table} as t`)
            .select(`t.${idColSelf}`)
            .where(sql`(${sourcePathSubquery})`, 'like', patternExpr)
            .where(`t.${idColSelf}`, '<>', effectiveRhs)
            .where(`t.deleted_at`, 'is', null);
        }

        return repositoryPredicateCall(eb, repositoryPredicateRef(eb, `${selfTable}.${idColSelf}`), 'in', subquery);
      }

      if ((lowerOp === 'child_of' || lowerOp === 'parent_of') && typeof fieldName === 'string') {
        if (!selfTable) throw new Error(`${lowerOp} requires selfTable`);

        const fieldMeta = meta.fields.get(fieldName) as FieldMetadata | undefined;
        if (!fieldMeta || fieldMeta.type !== 'ManyToOne' || !fieldMeta.relation) {
          throw new Error(`${meta.modelName || meta.className}.${fieldName} is not ManyToOne and cannot be used with ${lowerOp}`);
        }

        const relation = fieldMeta.relation as ManyToOneMetadata<BaseModel>;
        const targetCtor = relation.targetModel?.();
        if (!targetCtor) throw new Error(`unable to resolve targetModel for ${fieldName}`);
        const targetMeta = MetadataStorage.instance.getModelMetadata(targetCtor);
        const targetTable = targetMeta.tableName();
        if (!targetMeta.parentField) {
          throw new Error(`Target model ${targetMeta.modelName || targetMeta.className} does not configure parentField and cannot use ${lowerOp}`);
        }

        const fkCol = resolveStoredColumnName(fieldMeta, fieldName);
        const idField = targetMeta.fields.get('Id');
        const idCol = idField ? resolveStoredColumnName(idField, 'Id') : 'Id';
        const parentPathField = targetMeta.fields.get('ParentPath');
        const parentPathCol = parentPathField ? resolveStoredColumnName(parentPathField, 'ParentPath') : 'ParentPath';
        const dialect = String(getDialect() || 'postgres').toLowerCase();

        const sourcePathSubquery = db
          .selectFrom(`${targetTable} as s`)
          .select(`s.${parentPathCol}`)
          .where(`s.${idCol}`, '=', effectiveRhs)
          .where(`s.deleted_at`, 'is', null)
          .limit(1);

        let subquery: SelectSubqueryBuilder;
        if (lowerOp === 'child_of') {
          const patternExpr = dialect === 'mysql' ? sql`concat((${sourcePathSubquery}), '%')` : sql`(${sourcePathSubquery}) || '%'`;
          subquery = db
            .selectFrom(`${targetTable} as t`)
            .select(`t.${idCol}`)
            .where(`t.${parentPathCol}`, 'like', patternExpr)
            .where(`t.deleted_at`, 'is', null);
        } else {
          const patternExpr =
            dialect === 'mysql'
              ? sql`concat(${repositoryPredicateRef(eb, `t.${parentPathCol}`)}, '%')`
              : sql`${repositoryPredicateRef(eb, `t.${parentPathCol}`)} || '%'`;
          subquery = db
            .selectFrom(`${targetTable} as t`)
            .select(`t.${idCol}`)
            .where(sql`(${sourcePathSubquery})`, 'like', patternExpr)
            .where(`t.${idCol}`, '<>', effectiveRhs)
            .where(`t.deleted_at`, 'is', null);
        }

        let fkLhs: unknown;
        if (fieldMeta.companyDependent) {
          fkLhs = buildCompanyDependentFieldUnwrapExpr(dialect as DialectName, eb, `${selfTable}.${fkCol}`);
        } else {
          fkLhs = repositoryPredicateRef(eb, `${selfTable}.${fkCol}`);
        }
        return repositoryPredicateCall(eb, fkLhs, 'in', subquery);
      }

      if (typeof fieldName === 'string' && selfTable) {
        if (fieldName.includes('.')) {
          const leafMeta = resolveLeafMetaForPath(fieldName);
          const modelCtor = meta.type as unknown as ModelCtor<BaseModel>;
          const expr = createSelectCtx(selfTable).field(modelCtor, fieldName);
          const right = wrapByMeta(leafMeta, effectiveOp, effectiveRhs);
          return repositoryPredicateCall(eb, expr, effectiveOp, right);
        }

        const fieldMeta = meta.fields.get(fieldName);
        if (hasRepositorySqlComputeExpression(meta, fieldName)) {
          const expr = resolveRepositorySqlComputeExpression(meta, fieldName, createSelectCtx(selfTable));
          if (expr === undefined) {
            throw new Error(`field sql compute handler is missing: ${modelLabel}.${fieldName}`);
          }
          const right = wrapByMeta(fieldMeta, effectiveOp, effectiveRhs);
          return repositoryPredicateCall(eb, expr, effectiveOp, right);
        }

        if (fieldMeta?.translate) {
          const dialect = String(getDialect() || 'postgres') as DialectName;
          const unwrap = buildTranslatedFieldUnwrapExpr(dialect, eb, `${selfTable}.${fieldName}`);
          const right = wrapIfDecimal(fieldName, effectiveOp, effectiveRhs);
          const exact = repositoryPredicateCall(eb, unwrap, effectiveOp, right);
          return maybeAndTranslatedTrigramPrefilter(
            dialect,
            eb,
            selfTable,
            fieldName,
            fieldMeta,
            effectiveOp,
            effectiveRhs,
            exact
          );
        }

        if (fieldMeta && fieldMeta.companyDependent) {
          const rawDialect = getDialect();
          const dialect = (rawDialect ? String(rawDialect) : 'postgres') as DialectName;
          const col = resolveStoredColumnName(fieldMeta, fieldName);
          const unwrap = buildCompanyDependentFieldUnwrapExpr(dialect, eb, `${selfTable}.${col}`);
          const right = wrapIfDecimal(fieldName, effectiveOp, effectiveRhs);
          return repositoryPredicateCall(eb, unwrap, effectiveOp, right);
        }

        const right = wrapIfDecimal(fieldName, effectiveOp, effectiveRhs);
        return repositoryPredicateCall(eb, fieldName, effectiveOp, right);
      }

      const right = wrapIfDecimal(fieldName, effectiveOp, effectiveRhs);
      return repositoryPredicateCall(eb, lhs, effectiveOp, right);
    }

    if (typeof current === 'object' && current) {
      const currentNode = current as { And?: unknown; Or?: unknown };
      if (Array.isArray(currentNode.And)) {
        const parts = (currentNode.And as BaseQueryCondition[]).map(value => asPredicate(value));
        return repositoryPredicateAnd(eb, parts);
      }
      if (Array.isArray(currentNode.Or)) {
        const parts = (currentNode.Or as BaseQueryCondition[]).map(value => asPredicate(value));
        return repositoryPredicateOr(eb, parts);
      }
    }

    return repositoryPredicateAnd(eb, []);
  };

  return asPredicate(condition);
}
