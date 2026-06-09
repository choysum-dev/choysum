// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition, RepositoryConditionConverterLike, RepositoryCountAllDbLike, RepositoryExecuteUnknownQueryLike } from '../types';
import { asObjectRecord, asRuntimeCarrier } from '@/core/utils/object';

type RepositoryConditionQueryBuilder = {
  where(factory: (ctx: { eb: unknown }) => unknown): RepositoryConditionQueryBuilder;
};

type RepositoryConditionQuerySelectFrom = {
  select(selection: unknown): RepositoryConditionQueryBuilder;
};

type RepositoryConditionQueryDbLike = RepositoryCountAllDbLike<RepositoryConditionQuerySelectFrom, string>;

function asRepositoryConditionQueryDbLike(input: unknown): RepositoryConditionQueryDbLike {
  const record = asObjectRecord(input);
  const fnRecord = asRuntimeCarrier(record?.fn);
  if (!record || typeof record.selectFrom !== 'function' || !fnRecord || typeof fnRecord.countAll !== 'function') {
    throw new Error('repository condition query requires db.selectFrom and db.fn.countAll');
  }

  return input as RepositoryConditionQueryDbLike;
}

type RepositoryConditionQueryDeps = {
  db: unknown;
  table: string;
  applyConditionLayers: (condition: BaseQueryCondition) => BaseQueryCondition;
  isEmptyCondition: (condition: BaseQueryCondition) => boolean;
  convertCondition: RepositoryConditionConverterLike<BaseQueryCondition>;
  execute: RepositoryExecuteUnknownQueryLike;
};

export async function locateRepositoryIdsForCondition(params: RepositoryConditionQueryDeps, condition: BaseQueryCondition): Promise<string[]> {
  const db = asRepositoryConditionQueryDbLike(params.db);
  let query = db.selectFrom(params.table).select('Id');
  const filtered = params.applyConditionLayers(condition);
  if (!params.isEmptyCondition(filtered)) {
    query = query.where(({ eb }: { eb: unknown }) => params.convertCondition(eb, filtered, params.table));
  }

  const rows = (await params.execute<{ Id?: unknown }>(query)) || [];
  return rows.map(row => String(row?.Id || '')).filter(Boolean);
}

export async function countRepositoryConditionMatches(params: RepositoryConditionQueryDeps, condition: BaseQueryCondition): Promise<number> {
  const db = asRepositoryConditionQueryDbLike(params.db);
  let query = db.selectFrom(params.table).select(db.fn.countAll().as('Total'));
  const filtered = params.applyConditionLayers(condition);
  if (!params.isEmptyCondition(filtered)) {
    query = query.where(({ eb }: { eb: unknown }) => params.convertCondition(eb, filtered, params.table));
  }

  const rows = await params.execute<{ Total?: unknown }>(query);
  if (!rows?.length) return 0;
  const first = rows[0] as { Total?: unknown } | undefined;
  return Number(first?.Total) || 0;
}
