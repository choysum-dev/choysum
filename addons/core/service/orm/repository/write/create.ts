// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ConditionEnvelope, Entity, RepositoryExecute, RepositoryInsertIntoDbLike, RepositoryQueryLike } from '../types';
import {
  ensureRepositoryCreateAllowed,
  prepareRepositoryCreateEntities,
  type RepositoryCreateWriteAuthzDeps,
  type RepositoryCreateWritePrepareDeps,
} from './create_helpers';

export type RepositoryCreateWriteRuntimeDeps = {
  db: unknown;
  table: string;
  execute: RepositoryExecute;
  wrapSqlWriteError: (error: unknown, mode: 'create') => never;
};

type RepositoryCreateInsertBuilder = {
  values(rows: ReadonlyArray<Entity>): {
    returning(column: string): RepositoryQueryLike<{ Id: string }>;
  };
};

type RepositoryCreateDbLike = RepositoryInsertIntoDbLike<RepositoryCreateInsertBuilder, string>;

function asRepositoryCreateDbLike(input: unknown): RepositoryCreateDbLike {
  const record = input as { insertInto?: unknown } | null;
  if (!record || typeof record.insertInto !== 'function') {
    throw new Error('repository create requires db.insertInto');
  }

  return input as RepositoryCreateDbLike;
}

export type RepositoryCreateWritePostWriteDeps = {
  assertRecordRuleAllCreatedAllowed: (createdIds: string[], env: ConditionEnvelope) => Promise<void>;
  recomputePersistForCreate?: (createdIds: string[], sanitizedEntities: Entity[]) => Promise<void>;
};

type RepositoryCreateWriteDeps = RepositoryCreateWriteAuthzDeps &
  RepositoryCreateWritePrepareDeps &
  RepositoryCreateWriteRuntimeDeps &
  RepositoryCreateWritePostWriteDeps;

export async function insertRepositoryCreateEntities(params: RepositoryCreateWriteRuntimeDeps, entities: Entity[]): Promise<string[]> {
  const insertQuery = asRepositoryCreateDbLike(params.db).insertInto(params.table).values(entities).returning('Id');

  let result: Array<{ Id: string }> = [];
  try {
    result = await params.execute<{ Id: string }>(insertQuery);
  } catch (error) {
    params.wrapSqlWriteError(error, 'create');
  }

  if (!result?.length) {
    return [];
  }

  return result.map(row => row.Id);
}

export async function applyRepositoryCreatePostWrite(
  params: RepositoryCreateWritePostWriteDeps,
  ids: string[],
  recordRuleEnvelope: ConditionEnvelope
): Promise<string[]> {
  if (!ids.length) {
    return [];
  }

  await params.assertRecordRuleAllCreatedAllowed(ids, recordRuleEnvelope);
  return ids;
}

export async function executeRepositoryCreate(params: RepositoryCreateWriteDeps, value: Entity[]): Promise<string[]> {
  const recordRuleEnvelope = await ensureRepositoryCreateAllowed(params);
  const sanitizedEntities = await prepareRepositoryCreateEntities(params, value);
  const ids = await insertRepositoryCreateEntities(params, sanitizedEntities);
  const createdIds = await applyRepositoryCreatePostWrite(params, ids, recordRuleEnvelope);

  if (createdIds.length > 0 && typeof params.recomputePersistForCreate === 'function') {
    await params.recomputePersistForCreate(createdIds, sanitizedEntities);
  }

  return createdIds;
}
