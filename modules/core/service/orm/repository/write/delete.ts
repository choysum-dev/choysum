// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import { MetadataStorage, type ModelMetadata } from '../../metadata';
import { resolveManyToManyRelationJoinConfig, resolveOneToManyRelationConfig } from '../../relation/types';
import type {
  BaseQueryCondition,
  DeleteResult,
  RepositoryDeleteFromDbLike,
  RepositoryExecute,
  RepositoryQueryLike,
  RepositorySoftConditionPipelineDepsLike,
  RepositoryTableSoftConditionPipelineDepsLike,
  RepositoryUpdateTableDbLike,
  RepositoryWhereCapableLike,
} from '../types';
import type { RepositoryDeleteChild } from './delete_child_factory';
import {
  applyRepositoryDeleteCondition,
  resolveRepositoryDeleteTargetIds,
  type RepositoryDeleteWriteConditionDeps,
  type RepositoryDeleteWriteTargetDeps,
} from './delete_helpers';
import { asObjectRecord } from '../../../../utils/object';
import type { ObjectRecord } from '../../../../utils/types';

interface SoftDeleteUpdateQueryLike extends RepositoryWhereCapableLike<SoftDeleteUpdateQueryLike> {}

type RepositoryDeleteDbLike = RepositoryUpdateTableDbLike<SoftDeleteUpdateQueryLike, ObjectRecord, string> &
  RepositoryDeleteFromDbLike<RepositoryQueryLike, string>;

function resolveDeletePolicy(value: unknown): 'CASCADE' | 'SET NULL' | 'RESTRICT' | 'NO ACTION' {
  return value === 'CASCADE' || value === 'SET NULL' || value === 'RESTRICT' || value === 'NO ACTION' ? value : 'NO ACTION';
}

type RepositoryDeleteWriteDeps = RepositoryDeleteWriteTargetDeps &
  RepositoryDeleteWriteConditionDeps &
  RepositorySoftConditionPipelineDepsLike<BaseQueryCondition> & {
    db: unknown;
    softField: string;
    softDeleteEnabled: () => boolean;
    execute: RepositoryExecute;
    invalidateCache: () => void;
    wrapSqlWriteError: (error: unknown, mode: 'update') => never;
    createRepository: (meta: ModelMetadata) => RepositoryDeleteChild;
  };

export type RepositoryDeleteWriteRuntimeDeps = {
  execute: RepositoryExecute;
  wrapSqlWriteError: (error: unknown, mode: 'update') => never;
};

export type RepositoryDeleteWritePostWriteDeps = {
  invalidateCache: () => void;
};

export type RepositoryDeleteSoftDeletePreWriteDeps = {
  meta: ModelMetadata;
  db: unknown;
  softField: string;
  createRepository: (meta: ModelMetadata) => RepositoryDeleteChild;
} & RepositoryTableSoftConditionPipelineDepsLike<BaseQueryCondition>;

export type RepositoryDeleteQueryPrepareDeps = RepositoryDeleteWriteConditionDeps & {
  db: unknown;
  table: string;
};

export async function prepareRepositorySoftDeleteWrite(params: RepositoryDeleteSoftDeletePreWriteDeps, ids: string[]): Promise<RepositoryQueryLike> {
  const db = params.db as RepositoryDeleteDbLike;
  await handleRepositorySoftDeleteCascade(params, ids);

  const now = new Date();
  let query = db.updateTable(params.table).set({ [params.softField]: now, UpdatedAt: now });
  const softCond: BaseQueryCondition = params.applySoftLayer(['Id', 'in', ids]);
  if (!params.isEmptyCondition(softCond)) {
    query = query.where(({ eb }) => params.convertCondition(eb, softCond, params.table));
  }
  return query as unknown as RepositoryQueryLike;
}

export async function prepareRepositoryDeleteQuery(params: RepositoryDeleteQueryPrepareDeps, condition: BaseQueryCondition): Promise<RepositoryQueryLike> {
  const db = params.db as RepositoryDeleteDbLike;
  return await applyRepositoryDeleteCondition(db.deleteFrom(params.table), params, condition);
}

export async function executeRepositoryDeleteRuntime(
  params: RepositoryDeleteWriteRuntimeDeps,
  query: RepositoryQueryLike,
  wrapMode?: 'update'
): Promise<DeleteResult[]> {
  let rows: DeleteResult[] = [];
  if (!wrapMode) {
    rows = (await params.execute(query as RepositoryQueryLike<DeleteResult>)) || [];
    return rows;
  }

  try {
    rows = (await params.execute(query as RepositoryQueryLike<DeleteResult>)) || [];
  } catch (error) {
    params.wrapSqlWriteError(error, wrapMode);
  }
  return rows;
}

export function applyRepositoryDeletePostWrite(params: RepositoryDeleteWritePostWriteDeps, rows: DeleteResult[]): DeleteResult[] {
  if (rows && rows.length > 0) {
    params.invalidateCache();
  }
  return rows || [];
}

export async function executeRepositoryDelete(params: RepositoryDeleteWriteDeps, condition: BaseQueryCondition): Promise<DeleteResult[]> {
  const targetIds = await resolveRepositoryDeleteTargetIds(params, condition);
  if (!targetIds.length) return [];

  if (params.softDeleteEnabled()) {
    const query = await prepareRepositorySoftDeleteWrite(params, targetIds);
    const rows = await executeRepositoryDeleteRuntime(params, query);
    return applyRepositoryDeletePostWrite(params, rows);
  }

  const query = await prepareRepositoryDeleteQuery(params, condition);
  const rows = await executeRepositoryDeleteRuntime(params, query, 'update');
  return applyRepositoryDeletePostWrite(params, rows);
}

export async function executeRepositoryHardDelete(params: RepositoryDeleteWriteDeps, condition: BaseQueryCondition): Promise<DeleteResult[]> {
  const targetIds = await resolveRepositoryDeleteTargetIds(params, condition);
  if (!targetIds.length) return [];

  const query = await prepareRepositoryDeleteQuery(params, condition);
  const rows = await executeRepositoryDeleteRuntime(params, query);
  return applyRepositoryDeletePostWrite(params, rows);
}

async function handleRepositorySoftDeleteCascade(params: RepositoryDeleteSoftDeletePreWriteDeps, parentIds: string[]): Promise<void> {
  for (const [, fieldMeta] of params.meta.fields) {
    if (fieldMeta.type !== 'OneToMany') continue;

    const relation = resolveOneToManyRelationConfig(fieldMeta.relation);
    if (!relation) continue;

    const childMeta = MetadataStorage.instance.getModelMetadata(relation.targetModel());
    const foreignKeyField = relation.inverseField;
    const childForeignKeyMeta = childMeta.fields.get(foreignKeyField);
    const manyToOneRelation = asObjectRecord(childForeignKeyMeta?.relation);
    const policy = resolveDeletePolicy(manyToOneRelation?.onDelete);

    const childRepository = params.createRepository(childMeta);
    const foreignKeyCondition: BaseQueryCondition = [foreignKeyField, 'in', parentIds];

    switch (policy) {
      case 'CASCADE': {
        if (childRepository.softDeleteEnabled()) {
          await childRepository.delete(foreignKeyCondition);
        } else {
          await childRepository.hardDelete(foreignKeyCondition);
        }
        break;
      }
      case 'SET NULL': {
        if (childForeignKeyMeta?.column?.notNull) {
          throw new Error(`SET NULL blocked: ${childMeta.modelName}.${foreignKeyField} is NOT NULL`);
        }
        await childRepository.withFieldRuleBypass(async () => childRepository.update({ [foreignKeyField]: null }, foreignKeyCondition));
        break;
      }
      case 'RESTRICT':
      case 'NO ACTION':
      default: {
        const count = await childRepository.count(foreignKeyCondition);
        if (count > 0) {
          throw new Error(`Delete restricted by ${childMeta.modelName}: ${count} referencing record(s)`);
        }
      }
    }
  }

  for (const [, fieldMeta] of params.meta.fields) {
    if (fieldMeta.type !== 'ManyToMany') continue;

    const relation = resolveManyToManyRelationJoinConfig(fieldMeta.relation);
    if (!relation) continue;

    const joinMeta = MetadataStorage.instance.getModelMetadata(relation.joinModel());
    const joinRepository = params.createRepository(joinMeta);
    const joinCondition: BaseQueryCondition = [relation.joinField, 'in', parentIds];

    if (joinRepository.softDeleteEnabled()) {
      await joinRepository.delete(joinCondition);
    } else {
      await joinRepository.hardDelete(joinCondition);
    }
  }
}
