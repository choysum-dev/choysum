// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type {
  BaseQueryCondition,
  Entity,
  RepositoryExecute,
  RepositoryGetScalarFieldsDepsLike,
  RepositoryQueryLike,
  RepositoryRefBuilderLike,
  RepositorySoftConditionPipelineDepsLike,
  RepositorySelectionAliaserLike,
  RepositorySelectCallbackCapableLike,
  RepositorySelectCtxFactoryLike,
  RepositorySelectFromDbLike,
  RepositoryWhereCapableLike,
} from '../types';
import { hasRepositorySqlComputeExpression, resolveRepositorySqlComputeExpression } from '../query';
import {
  applyRepositoryMutationWriteCondition,
  resolveRepositoryMutationWriteTargetIds,
  type RepositoryMutationWriteConditionDeps,
  type RepositoryMutationWriteTargetDeps,
} from './mutation_write_helpers';
import type { ObjectRecord } from '../../../../utils/types';
import { withContext } from '../../../runtime/context/scope';

type RepositorySelectBuilderLike = RepositoryRefBuilderLike<string>;

interface RepositorySelectQueryLike
  extends
    RepositorySelectCallbackCapableLike<RepositorySelectQueryLike, RepositorySelectBuilderLike, unknown[]>,
    RepositoryWhereCapableLike<RepositorySelectQueryLike> {}

type RepositorySelectDbLike = RepositorySelectFromDbLike<RepositorySelectQueryLike, string>;

export type RepositoryUpdateWriteTargetDeps = RepositoryMutationWriteTargetDeps<'write'>;

export type RepositoryUpdateWriteConditionDeps = RepositoryMutationWriteConditionDeps<'write'>;

export type RepositoryUpdateWriteCurrentRowsDeps = {
  db: unknown;
  table: string;
  meta: ModelMetadata;
  makeSelectCtx: RepositorySelectCtxFactoryLike<ModelMetadata>;
  aliasSelection: RepositorySelectionAliaserLike;
  execute: RepositoryExecute;
  decodeFromDb: (row: Entity) => Entity;
} & RepositoryGetScalarFieldsDepsLike<ModelMetadata> &
  RepositorySoftConditionPipelineDepsLike<BaseQueryCondition>;

export async function resolveRepositoryUpdateTargetIds(params: RepositoryUpdateWriteTargetDeps, condition: BaseQueryCondition): Promise<string[]> {
  return await resolveRepositoryMutationWriteTargetIds(params, 'write', condition);
}

export async function applyRepositoryUpdateCondition<T>(query: T, params: RepositoryUpdateWriteConditionDeps, condition: BaseQueryCondition): Promise<T> {
  return await applyRepositoryMutationWriteCondition(query, params, 'write', condition);
}

export async function loadRepositoryUpdateValidationCurrentRows(
  params: RepositoryUpdateWriteCurrentRowsDeps,
  ids: string[]
): Promise<Map<string, ObjectRecord>> {
  const db = params.db as RepositorySelectDbLike;
  const normalizedIds = Array.from(new Set((ids || []).map(id => String(id || '').trim()).filter(Boolean)));
  if (normalizedIds.length === 0) {
    return new Map();
  }

  const scalarFields = Array.from(new Set(['Id', ...params.getScalarFields(params.meta)]));
  let query = db.selectFrom(params.table).select((builder: RepositorySelectBuilderLike) => {
    const selections: unknown[] = [];
    for (const field of scalarFields) {
      const fieldMeta = params.meta.fields.get(field);
      if (hasRepositorySqlComputeExpression(params.meta, field)) {
        const expr = resolveRepositorySqlComputeExpression(params.meta, field, params.makeSelectCtx(builder, params.table, params.meta));
        if (expr === undefined) {
          throw new Error(`field sql compute handler is missing: ${params.meta.fullModelName || params.meta.modelName || params.meta.name}.${field}`);
        }
        selections.push(params.aliasSelection(expr, field));
        continue;
      }

      selections.push(builder.ref(`${params.table}.${field}`).as(field));
    }
    return selections;
  });

  const filtered: BaseQueryCondition = params.applySoftLayer(['Id', 'in', normalizedIds]);
  if (!params.isEmptyCondition(filtered)) {
    query = query.where(({ eb }) => params.convertCondition(eb, filtered, params.table));
  }

  const rows = (await params.execute<Entity>(query as unknown as RepositoryQueryLike<Entity>)) || [];
  const result = new Map<string, ObjectRecord>();
  for (const row of rows) {
    // Prefetch lang / company maps so field updates can merge without wiping sibling keys.
    const decoded = withContext({ prefetch_langs: true, prefetch_companies: true }, () =>
      params.decodeFromDb(row)
    ) as ObjectRecord;
    const id = String(decoded.Id || '').trim();
    if (id) {
      result.set(id, decoded);
    }
  }

  return result;
}
