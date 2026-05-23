// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type {
  BaseQueryCondition,
  Entity,
  RepositoryExecute,
  RepositoryGetScalarFieldsDepsLike,
  RepositoryMutationPayloadGuardEncodeDepsLike,
  RepositoryMutationPayloadValidateDepsLike,
  RepositoryQueryLike,
  RepositorySoftConditionPipelineDepsLike,
  RepositorySelectionAliaserLike,
  RepositorySelectCtxFactoryLike,
  RepositoryUpdateTableDbLike,
  UpdateResult,
} from '../types';
import {
  applyRepositoryMutationDefaultValues,
  assertRepositoryMutationPayloadsAllowed,
  encodeRepositoryMutationPayloads,
  validateRepositoryMutationPayload,
} from './mutation_payload_helpers';
import {
  applyRepositoryUpdateCondition,
  loadRepositoryUpdateValidationCurrentRows,
  resolveRepositoryUpdateTargetIds,
  type RepositoryUpdateWriteConditionDeps,
  type RepositoryUpdateWriteCurrentRowsDeps,
  type RepositoryUpdateWriteTargetDeps,
} from './update_helpers';
import type { ObjectRecord } from '../../../../utils/types';
type RepositoryUpdateDbLike = RepositoryUpdateTableDbLike<unknown, ObjectRecord, string>;

export type RepositoryUpdateWriteRuntimeDeps = {
  execute: RepositoryExecute;
};

export type RepositoryUpdateWritePostWriteDeps = {
  invalidateCache: () => void;
  recomputePersistForUpdate?: (payload: { targetIds: string[]; sanitized: Entity; condition: BaseQueryCondition; rows: UpdateResult[] }) => Promise<void>;
};

type RepositoryUpdateWriteDeps = RepositoryUpdateWriteTargetDeps &
  RepositoryUpdateWriteConditionDeps &
  RepositoryUpdateWriteCurrentRowsDeps &
  RepositoryUpdateWriteRuntimeDeps &
  RepositoryUpdateWritePostWriteDeps & {
    db: unknown;
    makeSelectCtx: RepositorySelectCtxFactoryLike<ModelMetadata>;
    aliasSelection: RepositorySelectionAliaserLike;
    decodeFromDb: (row: Entity) => Entity;
    applyDefaultCompanyIdOnUpdate: (vals: Entity) => Entity;
  } & RepositoryGetScalarFieldsDepsLike<ModelMetadata> &
  RepositorySoftConditionPipelineDepsLike<BaseQueryCondition> &
  RepositoryMutationPayloadGuardEncodeDepsLike<Entity> &
  RepositoryMutationPayloadValidateDepsLike<Entity, 'update', ObjectRecord>;

type RepositoryPreparedUpdateWrite = {
  sanitized: Entity;
  targetIds: string[];
};

export type RepositoryUpdateWriteQueryPrepareDeps = RepositoryUpdateWriteConditionDeps & {
  db: unknown;
  table: string;
};

type RepositoryPreparedUpdateQuery = {
  query: RepositoryQueryLike<UpdateResult>;
};

export type RepositoryUpdateWriteTargetResolveDeps = RepositoryUpdateWriteTargetDeps;

export type RepositoryUpdateWriteSanitizedPayloadDeps = RepositoryUpdateWriteCurrentRowsDeps &
  RepositoryMutationPayloadGuardEncodeDepsLike<Entity> &
  RepositoryMutationPayloadValidateDepsLike<Entity, 'update', ObjectRecord> & {
    applyDefaultCompanyIdOnUpdate: (vals: Entity) => Entity;
  };

export async function resolveRepositoryUpdatePayloadTargets(
  params: RepositoryUpdateWriteTargetResolveDeps,
  condition: BaseQueryCondition
): Promise<string[] | undefined> {
  const targetIds = await resolveRepositoryUpdateTargetIds(params, condition);
  if (!targetIds.length) {
    return undefined;
  }
  return targetIds;
}

export async function prepareRepositoryUpdateSanitizedPayload(
  params: RepositoryUpdateWriteSanitizedPayloadDeps,
  vals: Entity,
  targetIds: string[]
): Promise<Entity> {
  await assertRepositoryMutationPayloadsAllowed(params, [vals]);

  const preparedVals = applyRepositoryMutationDefaultValues(
    {
      applyDefaultMutationValues: payload => params.applyDefaultCompanyIdOnUpdate(payload),
    },
    [vals]
  )[0] as Entity;
  const currentRows = await loadRepositoryUpdateValidationCurrentRows(params, targetIds);
  await validateRepositoryMutationPayload(
    {
      validateFields: (input, mode, current) => params.validateFields(input, mode, current),
    },
    preparedVals,
    'update',
    targetIds.map(id => currentRows.get(id))
  );

  return encodeRepositoryMutationPayloads(params, [preparedVals])[0] as Entity;
}

export async function prepareRepositoryUpdatePayload(
  params: Omit<RepositoryUpdateWriteDeps, keyof RepositoryUpdateWritePostWriteDeps>,
  vals: Entity,
  condition: BaseQueryCondition
): Promise<RepositoryPreparedUpdateWrite | undefined> {
  const targetIds = await resolveRepositoryUpdatePayloadTargets(params, condition);
  if (!targetIds) {
    return undefined;
  }

  const sanitized = await prepareRepositoryUpdateSanitizedPayload(params, vals, targetIds);
  return { sanitized, targetIds };
}

export async function prepareRepositoryUpdateQuery(
  params: RepositoryUpdateWriteQueryPrepareDeps,
  sanitized: Entity,
  condition: BaseQueryCondition
): Promise<RepositoryPreparedUpdateQuery> {
  const db = params.db as RepositoryUpdateDbLike;
  const updateQuery = db.updateTable(params.table).set(sanitized as ObjectRecord);
  const query = await applyRepositoryUpdateCondition(updateQuery, params, condition);
  return { query: query as RepositoryQueryLike<UpdateResult> };
}

export async function executeRepositoryUpdateRuntime(
  params: RepositoryUpdateWriteRuntimeDeps,
  query: RepositoryQueryLike<UpdateResult>
): Promise<UpdateResult[]> {
  const rows = await params.execute(query);
  return rows || [];
}

export function applyRepositoryUpdatePostWrite(params: RepositoryUpdateWritePostWriteDeps, rows: UpdateResult[]): UpdateResult[] {
  if (rows && rows.length > 0) {
    params.invalidateCache();
  }
  return rows || [];
}

export async function executeRepositoryUpdate(params: RepositoryUpdateWriteDeps, vals: Entity, condition: BaseQueryCondition): Promise<UpdateResult[]> {
  const preparedPayload = await prepareRepositoryUpdatePayload(params, vals, condition);
  if (!preparedPayload) {
    return [];
  }

  const preparedQuery = await prepareRepositoryUpdateQuery(params, preparedPayload.sanitized, condition);
  const rows = await executeRepositoryUpdateRuntime(params, preparedQuery.query);

  if (rows.length > 0 && typeof params.recomputePersistForUpdate === 'function') {
    await params.recomputePersistForUpdate({
      targetIds: preparedPayload.targetIds,
      sanitized: preparedPayload.sanitized,
      condition,
      rows,
    });
  }

  return applyRepositoryUpdatePostWrite(params, rows);
}
