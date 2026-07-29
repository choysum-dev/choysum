// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { getJsonHelpers } from './json_helpers';
import { buildHiddenScaleAlias } from '../hidden_scale_alias';
import {
  makeSelectCtx,
  type DbLike,
  normalizeOrderBy,
  resolveEffectiveOrder,
  applyOrderByToQuery,
  hasRepositorySqlComputeExpression,
  resolveRepositorySqlComputeExpression,
} from '../query';
import type BaseModel from '../../model/model';
import { FieldMetadata, ManyToManyMetadata, ManyToOneMetadata, ModelMetadata, OneToManyMetadata, type ModelCtor } from '../../metadata';
import { MetadataStorage } from '../../metadata';
import { REL_ALIAS_PREFIX } from '../../relation/relation_alias';
import { aliasSelection } from './selection_tree';
import { getRepositoryReadonlyCtx } from '../repository_runtime_bridge';
import type { SelectionNode, SelectionRelationEntry } from './selection_tree';
import { getRuntimeEnvFlag } from '@/core/utils/env';
import { asObjectRecord } from '../../../../utils/object';
import type { UnknownRecord } from '../../../../utils/types';
import { convertCondition } from '../query/condition_compiler';
import { isEmptyRepositoryCondition } from '../query/condition_layer';
import { resolveParentFieldRelationalCondition } from '../../model/model_for_field_condition';
import type { BaseQueryCondition } from '../types';

type RelationWhereBuilder = {
  (left: string, op: string, right: unknown): unknown;
  or: (clauses: unknown[]) => unknown;
};

type SelectBuilderWithRef = {
  ref: (path: string) => { as: (alias: string) => unknown };
};

type RelationPredicateWhereCapable<T> = T & {
  where: ((left: string, op: string, right: unknown) => T) & ((predicate: (eb: RelationWhereBuilder) => unknown) => T);
};

function toModelCtor(meta: ModelMetadata): ModelCtor<BaseModel> {
  return meta.type as unknown as ModelCtor<BaseModel>;
}

function getFieldSpec(fieldMeta: FieldMetadata | undefined): UnknownRecord {
  return asObjectRecord(fieldMeta?.column) ?? {};
}

function getRelationOrderBy(relation: unknown): unknown {
  return asObjectRecord(relation)?.orderBy;
}

export function applyRepositoryRelationSoftDeleteFilter<T extends { where: (left: string, op: string, right: unknown) => T }>(
  targetMeta: ModelMetadata,
  targetTable: string,
  subQuery: T
): T {
  const softField = 'DeletedAt';
  const globalEnabled = getRuntimeEnvFlag('CHOYSUM_SOFT_DELETE_ENABLED', true);
  const modelEnabled = targetMeta.softDelete ?? true;

  if (globalEnabled && modelEnabled) {
    return subQuery.where(`${targetTable}.${softField}`, 'is', null);
  }

  return subQuery;
}

/**
 * Apply parent field `@Field({ condition })` onto an O2M/M2M child subquery (PR-P1-F4 §5.5).
 */
export function applyRepositoryRelationFieldConditionFilter<
  T extends {
    where: ((left: string, op: string, right: unknown) => T) & ((predicate: (ctx: { eb: unknown }) => unknown) => T);
  },
>(
  db: DbLike,
  getDialect: () => string,
  parentMeta: ModelMetadata,
  relKey: string,
  targetMeta: ModelMetadata,
  targetTable: string,
  subQuery: T
): T {
  const condition = resolveParentFieldRelationalCondition(parentMeta, relKey);
  if (isEmptyRepositoryCondition(condition)) {
    return subQuery;
  }
  return subQuery.where(({ eb }: { eb: unknown }) =>
    convertCondition(db, getDialect, targetMeta, eb as never, condition as BaseQueryCondition, targetTable)
  );
}

function resolveRelationOwnershipField(targetMeta: ModelMetadata): string | undefined {
  const field = String(targetMeta.companyField ?? '').trim();
  if (!field) return undefined;
  if (!(targetMeta.fields instanceof Map) || !targetMeta.fields.has(field)) {
    throw new Error(
      `companyField model is missing ownership field (${targetMeta.fullModelName || targetMeta.modelName || targetMeta.name}: ${field})`
    );
  }
  return field;
}

function normalizeRepositoryRelationCompanyIds(): string[] {
  const ctx = asObjectRecord(getRepositoryReadonlyCtx());
  const raw = ctx?.enabledCompanyIds ?? ctx?.EnabledCompanyIds ?? ctx?.activeCompanyId ?? ctx?.ActiveCompanyId;

  const ids: string[] = [];
  if (Array.isArray(raw)) {
    for (const value of raw) {
      const normalized = String(value ?? '').trim();
      if (normalized) ids.push(normalized);
    }
  } else if (raw != null) {
    const normalized = String(raw ?? '').trim();
    if (normalized) ids.push(normalized);
  }

  return [...new Set(ids)];
}

export function applyRepositoryRelationCompanyFilter<T extends { where: (...args: unknown[]) => T }>(
  targetMeta: ModelMetadata,
  targetTable: string,
  subQuery: T
): T {
  const globalEnabled = getRuntimeEnvFlag('CHOYSUM_GRPC_COMPANY_FILTER_ENABLED', true);
  if (!globalEnabled) return subQuery;

  const ownershipField = resolveRelationOwnershipField(targetMeta);
  if (!ownershipField) return subQuery;

  const companyIds = normalizeRepositoryRelationCompanyIds();
  if (!companyIds.length) {
    throw new Error('missing ctx.enabledCompanyIds/activeCompanyId for company scoped operation');
  }

  const whereCapable = subQuery as RelationPredicateWhereCapable<T>;
  return whereCapable.where((eb: RelationWhereBuilder) =>
    eb.or([
      eb(`${targetTable}.${ownershipField}`, 'in', companyIds),
      eb(`${targetTable}.${ownershipField}`, 'is', null),
    ])
  );
}

type RepositoryRelationChildSelectDeps = {
  buildRelationJsonSelect: (db: DbLike, getDialect: () => string, parentMeta: ModelMetadata, relKey: string, entry: SelectionRelationEntry) => unknown;
};

export function buildRepositoryRelationChildSelect(
  db: DbLike,
  getDialect: () => string,
  targetMeta: ModelMetadata,
  targetTable: string,
  node: SelectionNode,
  deps: RepositoryRelationChildSelectDeps
) {
  return (sqb: unknown) => {
    const builder = sqb as SelectBuilderWithRef;
    const selections: unknown[] = [];
    const scalarFields = Array.from(node.columns);
    if (!scalarFields.includes('Id')) scalarFields.unshift('Id');

    for (const field of scalarFields) {
      const fieldMeta = targetMeta.fields.get(field) as FieldMetadata | undefined;
      if (field.includes('.')) {
        const expr = makeSelectCtx(db, getDialect, builder, targetTable, targetMeta).field(toModelCtor(targetMeta), field);
        selections.push(aliasSelection(expr, field));
      } else if (hasRepositorySqlComputeExpression(targetMeta, field)) {
        const out = resolveRepositorySqlComputeExpression(targetMeta, field, makeSelectCtx(db, getDialect, builder, targetTable, targetMeta));
        if (out === undefined) {
          throw new Error(`field sql compute handler is missing: ${targetMeta.fullModelName || targetMeta.modelName || targetMeta.name}.${field}`);
        }
        selections.push(aliasSelection(out, field));
      } else {
        selections.push(builder.ref(`${targetTable}.${field}`).as(field));
      }

      if (fieldMeta?.type === 'decimal') {
        const spec = getFieldSpec(fieldMeta);
        const scaleField = typeof spec.scaleField === 'string' ? spec.scaleField : undefined;
        if (typeof spec.scale !== 'number' && scaleField) {
          const scaleMeta = targetMeta.fields.get(scaleField);
          if (hasRepositorySqlComputeExpression(targetMeta, scaleField)) {
            const out = resolveRepositorySqlComputeExpression(targetMeta, scaleField, makeSelectCtx(db, getDialect, builder, targetTable, targetMeta));
            if (out === undefined) {
              throw new Error(`field sql compute handler is missing: ${targetMeta.fullModelName || targetMeta.modelName || targetMeta.name}.${scaleField}`);
            }
            selections.push(aliasSelection(out, buildHiddenScaleAlias(field)));
          } else if (scaleMeta?.column) {
            selections.push(builder.ref(`${targetTable}.${scaleField}`).as(buildHiddenScaleAlias(field)));
          }
        }
      }
    }

    for (const [childRelKey, childRelEntry] of node.relations) {
      const expr = deps.buildRelationJsonSelect(db, getDialect, targetMeta, childRelKey, childRelEntry);
      if (expr) selections.push(aliasSelection(expr, `${REL_ALIAS_PREFIX}${childRelKey}`));
    }

    return selections;
  };
}

export function buildRelationJsonSelect(
  db: DbLike,
  getDialect: () => string,
  parentMeta: ModelMetadata,
  relKey: string,
  entry: SelectionRelationEntry
): unknown {
  const { jsonArrayFrom, jsonObjectFrom } = getJsonHelpers(getDialect() as Parameters<typeof getJsonHelpers>[0]);
  const parentTable = parentMeta.tableName();

  if (entry.fieldType === 'ManyToOne') {
    const relation = entry.relation as ManyToOneMetadata<BaseModel>;
    const targetMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());
    const targetTable = targetMeta.tableName();
    const targetRef = targetTable === parentTable ? `${targetTable}__rel_${relKey}` : targetTable;
    const targetFrom = targetRef === targetTable ? targetTable : `${targetTable} as ${targetRef}`;

    let sub = db
      .selectFrom(targetFrom)
      .select(buildRepositoryRelationChildSelect(db, getDialect, targetMeta, targetRef, entry.node, { buildRelationJsonSelect }))
      .whereRef(`${targetRef}.Id`, '=', `${parentTable}.${relKey}`);

    sub = applyRepositoryRelationCompanyFilter(targetMeta, targetRef, sub);
    sub = applyRepositoryRelationSoftDeleteFilter(targetMeta, targetRef, sub);
    return jsonObjectFrom(sub as unknown as Parameters<typeof jsonObjectFrom>[0]);
  }

  if (entry.fieldType === 'OneToMany') {
    const relation = entry.relation as OneToManyMetadata<BaseModel>;
    const targetMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());
    const targetTable = targetMeta.tableName();
    const inverseField = relation.inverseField;
    const targetRef = targetTable === parentTable ? `${targetTable}__rel_${relKey}` : targetTable;
    const targetFrom = targetRef === targetTable ? targetTable : `${targetTable} as ${targetRef}`;

    let sub = db
      .selectFrom(targetFrom)
      .select(buildRepositoryRelationChildSelect(db, getDialect, targetMeta, targetRef, entry.node, { buildRelationJsonSelect }))
      .whereRef(`${targetRef}.${inverseField}`, '=', `${parentTable}.Id`);

    sub = applyRepositoryRelationCompanyFilter(targetMeta, targetRef, sub);
    sub = applyRepositoryRelationSoftDeleteFilter(targetMeta, targetRef, sub);
    sub = applyRepositoryRelationFieldConditionFilter(db, getDialect, parentMeta, relKey, targetMeta, targetRef, sub as never);

    const relationOrder = normalizeOrderBy(getRelationOrderBy(relation));
    const metaOrder = normalizeOrderBy(targetMeta.orderBy);
    const effectiveOrder = resolveEffectiveOrder(relationOrder, metaOrder, targetMeta);
    sub = applyOrderByToQuery(sub, targetMeta, targetRef, effectiveOrder, {
      getDialect,
      resolvePathField(builder, field) {
        const ctx = makeSelectCtx(db, getDialect, builder, targetRef, targetMeta);
        return ctx.field(toModelCtor(targetMeta), field);
      },
      resolveSelectField(builder, _field, _fieldMeta) {
        if (!hasRepositorySqlComputeExpression(targetMeta, _field)) return undefined;
        return resolveRepositorySqlComputeExpression(targetMeta, _field, makeSelectCtx(db, getDialect, builder, targetRef, targetMeta));
      },
    });

    return jsonArrayFrom(sub as unknown as Parameters<typeof jsonArrayFrom>[0]);
  }

  if (entry.fieldType === 'ManyToMany') {
    const relation = entry.relation as ManyToManyMetadata<BaseModel, BaseModel>;
    const targetMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());
    const targetTable = targetMeta.tableName();
    const targetRef = targetTable === parentTable ? `${targetTable}__rel_${relKey}` : targetTable;
    const targetFrom = targetRef === targetTable ? targetTable : `${targetTable} as ${targetRef}`;
    const joinMeta = MetadataStorage.instance.getModelMetadata(relation.joinModel());
    const joinTable = joinMeta.tableName();
    const joinField = relation.joinField;
    const inverseJoinField = relation.inverseJoinField;

    let sub = db
      .selectFrom(joinTable)
      .innerJoin(targetFrom, `${targetRef}.Id`, `${joinTable}.${inverseJoinField}`)
      .select(buildRepositoryRelationChildSelect(db, getDialect, targetMeta, targetRef, entry.node, { buildRelationJsonSelect }))
      .whereRef(`${joinTable}.${joinField}`, '=', `${parentTable}.Id`);

    sub = applyRepositoryRelationCompanyFilter(joinMeta, joinTable, sub);
    sub = applyRepositoryRelationCompanyFilter(targetMeta, targetRef, sub);
    sub = applyRepositoryRelationSoftDeleteFilter(joinMeta, joinTable, sub);
    sub = applyRepositoryRelationSoftDeleteFilter(targetMeta, targetRef, sub);
    sub = applyRepositoryRelationFieldConditionFilter(db, getDialect, parentMeta, relKey, targetMeta, targetRef, sub as never);

    const relationOrder = normalizeOrderBy(getRelationOrderBy(relation));
    const metaOrder = normalizeOrderBy(targetMeta.orderBy);
    const effectiveOrder = resolveEffectiveOrder(relationOrder, metaOrder, targetMeta);
    sub = applyOrderByToQuery(sub, targetMeta, targetRef, effectiveOrder, {
      getDialect,
      resolvePathField(builder, field) {
        const ctx = makeSelectCtx(db, getDialect, builder, targetRef, targetMeta);
        return ctx.field(toModelCtor(targetMeta), field);
      },
      resolveSelectField(builder, _field, _fieldMeta) {
        if (!hasRepositorySqlComputeExpression(targetMeta, _field)) return undefined;
        return resolveRepositorySqlComputeExpression(targetMeta, _field, makeSelectCtx(db, getDialect, builder, targetRef, targetMeta));
      },
    });

    return jsonArrayFrom(sub as unknown as Parameters<typeof jsonArrayFrom>[0]);
  }

  return null;
}
