// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { BaseQueryCondition, RepositoryRecordRuleConditionPipelineDepsLike } from '../types';

export type RepositoryMutationWriteOp = 'delete' | 'write';

export type RepositoryMutationWriteTargetDeps<TOp extends RepositoryMutationWriteOp> = {
  meta: ModelMetadata;
  locateIdsForCondition: (condition: BaseQueryCondition) => Promise<string[]>;
  assertCompanyWriteAccessForCondition: (condition: BaseQueryCondition) => Promise<string[]>;
  assertRecordRuleAllTargetsAllowed: (op: TOp, targetIds: string[]) => Promise<void>;
};

export type RepositoryMutationWriteConditionDeps<TOp extends RepositoryMutationWriteOp> = {
  table: string;
} & RepositoryRecordRuleConditionPipelineDepsLike<TOp, BaseQueryCondition>;

type RepositoryWhereQueryLike<TQuery = unknown> = TQuery & {
  where: (predicate: (args: { eb: unknown }) => unknown) => TQuery;
};

export async function resolveRepositoryMutationWriteTargetIds<TOp extends RepositoryMutationWriteOp>(
  params: RepositoryMutationWriteTargetDeps<TOp>,
  op: TOp,
  condition: BaseQueryCondition
): Promise<string[]> {
  const targetIds = params.meta.companyScoped ? await params.assertCompanyWriteAccessForCondition(condition) : await params.locateIdsForCondition(condition);

  if (!targetIds.length) {
    return [];
  }

  await params.assertRecordRuleAllTargetsAllowed(op, targetIds);
  return targetIds;
}

export async function applyRepositoryMutationWriteCondition<T, TOp extends RepositoryMutationWriteOp>(
  query: T,
  params: RepositoryMutationWriteConditionDeps<TOp>,
  op: TOp,
  condition: BaseQueryCondition
): Promise<T> {
  const condWithRR = await params.applyRecordRuleToCondition(condition, op);
  const filtered = params.applyDefaultLayers(condWithRR);
  if (params.isEmptyCondition(filtered)) {
    return query;
  }

  const whereCapable = query as RepositoryWhereQueryLike<T>;
  return whereCapable.where(({ eb }) => params.convertCondition(eb, filtered, params.table));
}
