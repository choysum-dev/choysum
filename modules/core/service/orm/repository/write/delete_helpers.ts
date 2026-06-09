// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { BaseQueryCondition } from '../types';
import {
  applyRepositoryMutationWriteCondition,
  resolveRepositoryMutationWriteTargetIds,
  type RepositoryMutationWriteConditionDeps,
  type RepositoryMutationWriteTargetDeps,
} from './mutation_write_helpers';

export type RepositoryDeleteWriteTargetDeps = RepositoryMutationWriteTargetDeps<'delete'>;

export type RepositoryDeleteWriteConditionDeps = RepositoryMutationWriteConditionDeps<'delete'>;

export async function resolveRepositoryDeleteTargetIds(params: RepositoryDeleteWriteTargetDeps, condition: BaseQueryCondition): Promise<string[]> {
  return await resolveRepositoryMutationWriteTargetIds(params, 'delete', condition);
}

export async function applyRepositoryDeleteCondition<T>(query: T, params: RepositoryDeleteWriteConditionDeps, condition: BaseQueryCondition): Promise<T> {
  return await applyRepositoryMutationWriteCondition(query, params, 'delete', condition);
}
