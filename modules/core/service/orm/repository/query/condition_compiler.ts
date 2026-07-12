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
import { rewriteSearchCondition } from '../../../runtime/compute/search_rewrite';

function supportsContainsFieldType(fieldMeta: FieldMetadata | undefined): boolean {
  const fieldType = fieldMeta?.type;
  return fieldType === 'jsonobject' || fieldType === 'ManyToManyRef';
}

export function convertCondition(
  db: DbLike,
  getDialect: () => string,
  meta: ModelMetadata,
  eb: RepositoryPredicateBuilder,
  condition: BaseQueryCondition,
  selfTable?: string
): RepositoryPredicate {
  const createSelectCtx = (table: string) => createRepositoryPredicateSelectCtx(db, getDialect, eb, table, meta);

  const wrapIfDecimal = (fieldName: string | undefined, op: unknown, rhs: unknown) => {
    if (!fieldName) return rhs;
    const fieldMeta = meta.fields.get(fieldName);
    if (fieldMeta?.type !== 'decimal') return rhs;

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
    if (!fieldMeta || fieldMeta.type !== 'decimal') return rhs;
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
            const fieldMeta = meta.fields.get(fieldName);
            if (fieldMeta?.select) {
              lhsExpr = fieldMeta.select!.expr(createSelectCtx(selfTable));
            } else {
              lhsExpr = repositoryPredicateRef(eb, `${selfTable}.${fieldName}`);
            }
          }
        } else {
          lhsExpr = repositoryPredicateRef(eb, lhs);
        }

        const pattern = typeof effectiveRhs === 'string' ? effectiveRhs : String(effectiveRhs ?? '');
        if (String(dialect).toLowerCase() === 'postgres') {
          return repositoryPredicateCall(eb, lhsExpr, effectiveOp, pattern);
        }

        const mapped = lowerOp === 'not ilike' ? 'not like' : 'like';
        return repositoryPredicateCall(eb, lower(lhsExpr as Parameters<typeof lower>[0]), mapped, pattern.toLowerCase());
      }

      if (lowerOp === 'contains') {
        const dialect = (getDialect() as DialectName) || 'postgres';

        if (typeof fieldName === 'string') {
          const fieldMeta = meta.fields.get(fieldName);
          if (fieldMeta && !supportsContainsFieldType(fieldMeta)) {
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
            const fieldMeta = meta.fields.get(fieldName);
            if (fieldMeta?.select) {
              lhsExpr = fieldMeta.select!.expr(createSelectCtx(selfTable));
            } else {
              lhsExpr = repositoryPredicateRef(eb, `${selfTable}.${fieldName}`);
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
        const idColumn = meta.fields.get('Id')?.column as unknown;
        const idColSelf =
          typeof idColumn === 'object' && idColumn !== null && 'name' in idColumn && typeof (idColumn as { name?: unknown }).name === 'string'
            ? (idColumn as { name: string }).name
            : 'Id';
        const parentPathColumn = meta.fields.get('ParentPath')?.column as unknown;
        const parentPathCol =
          typeof parentPathColumn === 'object' &&
          parentPathColumn !== null &&
          'name' in parentPathColumn &&
          typeof (parentPathColumn as { name?: unknown }).name === 'string'
            ? (parentPathColumn as { name: string }).name
            : 'ParentPath';
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

        const fkColumn = fieldMeta?.column as unknown;
        const fkCol =
          typeof fkColumn === 'object' && fkColumn !== null && 'name' in fkColumn && typeof (fkColumn as { name?: unknown }).name === 'string'
            ? (fkColumn as { name: string }).name
            : fieldName;
        const idColumn = targetMeta.fields.get('Id')?.column as unknown;
        const idCol =
          typeof idColumn === 'object' && idColumn !== null && 'name' in idColumn && typeof (idColumn as { name?: unknown }).name === 'string'
            ? (idColumn as { name: string }).name
            : 'Id';
        const parentPathColumn = targetMeta.fields.get('ParentPath')?.column as unknown;
        const parentPathCol =
          typeof parentPathColumn === 'object' &&
          parentPathColumn !== null &&
          'name' in parentPathColumn &&
          typeof (parentPathColumn as { name?: unknown }).name === 'string'
            ? (parentPathColumn as { name: string }).name
            : 'ParentPath';
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

        return repositoryPredicateCall(eb, repositoryPredicateRef(eb, `${selfTable}.${fkCol}`), 'in', subquery);
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
        if (fieldMeta?.select) {
          const expr = fieldMeta.select!.expr(createSelectCtx(selfTable));
          const right = wrapByMeta(fieldMeta, effectiveOp, effectiveRhs);
          return repositoryPredicateCall(eb, expr, effectiveOp, right);
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
