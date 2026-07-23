// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import BaseModel from '../../model/model';
import { getStringHelpers } from './string_helpers';
import { FieldMetadata, ManyToOneMetadata, MetadataStorage, ModelCtor, ModelMetadata } from '../../metadata';
import type { SelectCtx, SelectExpressionAtom, SelectExpressionValue, SelectSubqueryBuilder } from '../../metadata';
import type { DialectName } from '../repository_dialect';
import type { ExpressionBuilder } from '../types';
import { hasRepositorySqlComputeExpression, isRepositorySelectableScalarField, resolveRepositorySqlComputeExpression } from './sql_compute_expression';
import { buildTranslatedFieldUnwrapExpr } from './translated_field_sql';
import type { ObjectRecord } from '../../../../utils/types';

export interface DbLike {
  selectFrom(table: string): SelectSubqueryBuilder;
}

export function makeSelectCtx(db: DbLike, getDialect: () => string, builder: unknown, selfTable: string, curMeta: ModelMetadata): SelectCtx {
  const expressionBuilder = builder as ExpressionBuilder<ObjectRecord, string>;
  const refBuilder = builder as { ref(path: string): SelectExpressionAtom };
  const str = getStringHelpers(getDialect() as DialectName, expressionBuilder);

  const resolveMetaAndTable = (model: ModelCtor<BaseModel>) => {
    const meta = MetadataStorage.instance.getModelMetadata(model);
    return { meta, table: meta.tableName() };
  };

  const buildManyToOnePathSubquery = (rootTable: string, rootMeta: ModelMetadata, segments: string[]): SelectExpressionValue => {
    type Hop = {
      table: string;
      meta: ModelMetadata;
      fieldName: string;
      target: { table: string; meta: ModelMetadata };
    };

    const hops: Hop[] = [];
    let currentMeta = rootMeta;
    let currentTable = rootTable;

    for (const segment of segments.slice(0, -1)) {
      const fieldMeta = currentMeta.fields.get(segment) as FieldMetadata | undefined;
      if (!fieldMeta) throw new Error(`field(${currentMeta.type.name}.${segment}) not found`);
      if (fieldMeta.type !== 'ManyToOne' || !fieldMeta.relation || !fieldMeta.column) {
        throw new Error(`path segment ${segment} must be a ManyToOne field with a column`);
      }

      const targetCtor = (fieldMeta.relation as ManyToOneMetadata<BaseModel>).targetModel?.();
      if (!targetCtor) throw new Error(`unable to resolve targetModel for ${segment}`);
      const targetMeta = MetadataStorage.instance.getModelMetadata(targetCtor);
      const targetTable = targetMeta.tableName();

      hops.push({
        table: currentTable,
        meta: currentMeta,
        fieldName: segment,
        target: { table: targetTable, meta: targetMeta },
      });

      currentMeta = targetMeta;
      currentTable = targetTable;
    }

    const finalField = segments[segments.length - 1];
    const finalMeta = currentMeta;
    const finalTable = currentTable;
    const finalFieldMeta = finalMeta.fields.get(finalField) as FieldMetadata | undefined;
    if (!finalFieldMeta) throw new Error(`field(${finalMeta.type.name}.${finalField}) not found`);

    const canShortcutIdTail = finalField === 'Id' && Boolean(finalFieldMeta.column) && !hasRepositorySqlComputeExpression(finalMeta, finalField);
    if (canShortcutIdTail && hops.length === 1) {
      // A single-hop ManyToOne Id path can reuse the root foreign-key column directly.
      return refBuilder.ref(`${rootTable}.${hops[0].fieldName}`);
    }

    if (canShortcutIdTail && hops.length > 1) {
      // A multi-hop ManyToOne Id tail can read the prior hop foreign key and skip the terminal table.
      const idSourceHop = hops[hops.length - 1];
      const idSourceTable = idSourceHop.table;

      let subquery = db.selectFrom(idSourceTable);
      let rightTable = idSourceTable;
      for (let index = hops.length - 2; index >= 1; index--) {
        const hop = hops[index];
        subquery = subquery.innerJoin(hop.table, `${hop.table}.${hop.fieldName}`, `${rightTable}.Id`);
        rightTable = hop.table;
      }

      subquery = subquery.select(`${idSourceTable}.${idSourceHop.fieldName}`);

      const firstHop = hops[0];
      const anchorTable = hops.length >= 2 ? hops[1].table : idSourceTable;
      subquery = subquery.whereRef(`${anchorTable}.Id`, '=', `${rootTable}.${firstHop.fieldName}`);

      return subquery;
    }

    let subquery = db.selectFrom(finalTable);
    let rightTable = finalTable;
    for (let index = hops.length - 1; index >= 1; index--) {
      const hop = hops[index];
      subquery = subquery.innerJoin(hop.table, `${hop.table}.${hop.fieldName}`, `${rightTable}.Id`);
      rightTable = hop.table;
    }

    if (hasRepositorySqlComputeExpression(finalMeta, finalField)) {
      subquery = subquery.select((subBuilder: unknown) => {
        const targetCtx = makeSelectCtx(db, getDialect, subBuilder, finalTable, finalMeta);
        const resolved = resolveRepositorySqlComputeExpression(finalMeta, finalField, targetCtx);
        if (resolved === undefined) {
          throw new Error(`field(${finalMeta.type.name}.${finalField}) sql compute handler is missing`);
        }
        return resolved;
      });
    } else if (finalFieldMeta.column) {
      // Translate fields store lang maps; path expressions must unwrap to text for LIKE/order/display.
      if (finalFieldMeta.translate) {
        subquery = subquery.select((subBuilder: unknown) =>
          buildTranslatedFieldUnwrapExpr(getDialect() as DialectName, subBuilder as any, `${finalTable}.${finalField}`)
        );
      } else {
        subquery = subquery.select(`${finalTable}.${finalField}`);
      }
    } else {
      throw new Error(`field(${finalMeta.type.name}.${finalField}) has neither sql compute handler nor column`);
    }

    if (hops.length > 0) {
      const firstHop = hops[0];
      const anchorTable = hops.length >= 2 ? hops[1].table : finalTable;
      subquery = subquery.whereRef(`${anchorTable}.Id`, '=', `${rootTable}.${firstHop.fieldName}`);
    }

    return subquery;
  };

  const fieldImpl = (model: ModelCtor<BaseModel>, path: string): SelectExpressionValue => {
    const { meta, table } = resolveMetaAndTable(model);
    const parts = path.split('.');

    if (parts.length === 1) {
      const fieldName = parts[0];
      const fieldMeta = meta.fields.get(fieldName) as FieldMetadata | undefined;
      if (!fieldMeta) throw new Error(`field(${model.name}.${fieldName}) not found`);

      if (hasRepositorySqlComputeExpression(meta, fieldName)) {
        const targetCtx = makeSelectCtx(db, getDialect, builder, table, meta);
        const resolved = resolveRepositorySqlComputeExpression(meta, fieldName, targetCtx);
        if (resolved === undefined) {
          throw new Error(`field(${model.name}.${fieldName}) sql compute handler is missing`);
        }
        return resolved as SelectExpressionValue;
      }

      if (fieldMeta.column) {
        // Data-i18n: `$sql.field('Name')` (e.g. DisplayName SqlCompute) must unwrap jsonb lang maps
        // so keyword `like`/`ilike` predicates do not become `jsonb ~~ text`.
        if (fieldMeta.translate) {
          return buildTranslatedFieldUnwrapExpr(getDialect() as DialectName, expressionBuilder as any, `${table}.${fieldName}`) as SelectExpressionValue;
        }
        return refBuilder.ref(`${table}.${fieldName}`);
      }
      throw new Error(`field(${model.name}.${fieldName}) has neither sql compute handler nor column`);
    }

    let currentMeta = meta;
    for (const segment of parts.slice(0, -1)) {
      const fieldMeta = currentMeta.fields.get(segment) as FieldMetadata | undefined;
      if (!fieldMeta || fieldMeta.type !== 'ManyToOne' || !fieldMeta.relation) {
        throw new Error(`path ${path} only supports ManyToOne chains; intermediate segment ${segment} is not ManyToOne`);
      }
      const targetCtor = (fieldMeta.relation as ManyToOneMetadata<BaseModel>).targetModel?.();
      if (!targetCtor) throw new Error(`unable to resolve targetModel for ${segment}`);
      currentMeta = MetadataStorage.instance.getModelMetadata(targetCtor);
    }

    return buildManyToOnePathSubquery(table, meta, parts);
  };

  const fieldExistImpl = (model: ModelCtor<BaseModel>, path: string) => {
    const { meta } = resolveMetaAndTable(model);
    const parts = path.split('.');
    let currentMeta = meta;

    for (let index = 0; index < parts.length; index++) {
      const segment = parts[index];
      const fieldMeta = currentMeta.fields.get(segment) as FieldMetadata | undefined;
      if (!fieldMeta) return false;

      const isLeaf = index === parts.length - 1;
      if (isLeaf) {
        return isRepositorySelectableScalarField(currentMeta, segment, fieldMeta);
      }

      if (fieldMeta.type !== 'ManyToOne' || !fieldMeta.relation) return false;
      const targetCtor = (fieldMeta.relation as ManyToOneMetadata<BaseModel>).targetModel();
      currentMeta = MetadataStorage.instance.getModelMetadata(targetCtor);
    }

    return false;
  };

  const currentModelCtor = (curMeta.type as unknown as ModelCtor<BaseModel>) || (BaseModel as unknown as ModelCtor<BaseModel>);

  const resolveFieldArgs = (modelOrPath: unknown, path: unknown): { model: ModelCtor<BaseModel>; fieldPath: string } => {
    if (typeof path === 'string') {
      return {
        model: modelOrPath as ModelCtor<BaseModel>,
        fieldPath: String(path),
      };
    }

    return {
      model: currentModelCtor,
      fieldPath: String(modelOrPath ?? ''),
    };
  };

  const fieldResolver: SelectCtx['field'] = ((modelOrPath: unknown, path?: unknown) => {
    const resolved = resolveFieldArgs(modelOrPath, path);
    return fieldImpl(resolved.model, resolved.fieldPath);
  }) as SelectCtx['field'];

  const fieldExistResolver: SelectCtx['fieldExist'] = ((modelOrPath: unknown, path?: unknown) => {
    const resolved = resolveFieldArgs(modelOrPath, path);
    return fieldExistImpl(resolved.model, resolved.fieldPath);
  }) as SelectCtx['fieldExist'];

  const ctx: SelectCtx = {
    eb: expressionBuilder,
    col: (table: string, column: string) => refBuilder.ref(`${table}.${column}`),
    field: fieldResolver,
    fieldExist: fieldExistResolver,
    model: currentModelCtor,
    str,
    selectFrom: (table: string) => db.selectFrom(table),
  };
  void selfTable;
  return ctx;
}
