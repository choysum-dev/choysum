// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ModelMetadata } from '../../metadata';
import type { ExpressionBuilder, ExpressionWrapper } from '../types';
import type { SqlBool } from 'kysely';
import { makeSelectCtx, type DbLike } from './select_context';
import type { ObjectRecord } from '../../../../utils/types';

export type RepositoryPredicateBuilder = ExpressionBuilder<ObjectRecord, string>;
export type RepositoryPredicate = ExpressionWrapper<ObjectRecord, string, SqlBool>;

type RepositoryPredicateBuilderLike = RepositoryPredicateBuilder & {
  (lhs: unknown, operator: unknown, rhs: unknown): RepositoryPredicate;
  ref(path: string): unknown;
  and(parts: readonly RepositoryPredicate[]): RepositoryPredicate;
  or(parts: readonly RepositoryPredicate[]): RepositoryPredicate;
};

function asRepositoryPredicateBuilderLike(builder: RepositoryPredicateBuilder): RepositoryPredicateBuilderLike {
  return builder as unknown as RepositoryPredicateBuilderLike;
}

export function createRepositoryPredicateSelectCtx(
  db: DbLike,
  getDialect: () => string,
  builder: RepositoryPredicateBuilder,
  selfTable: string,
  meta: ModelMetadata
) {
  return makeSelectCtx(db, getDialect, builder, selfTable, meta);
}

export function repositoryPredicateRef(builder: RepositoryPredicateBuilder, path: string): unknown {
  return asRepositoryPredicateBuilderLike(builder).ref(path);
}

export function repositoryPredicateCall(builder: RepositoryPredicateBuilder, lhs: unknown, operator: unknown, rhs: unknown): RepositoryPredicate {
  return asRepositoryPredicateBuilderLike(builder)(lhs, operator, rhs);
}

export function repositoryPredicateAnd(builder: RepositoryPredicateBuilder, parts: RepositoryPredicate[]): RepositoryPredicate {
  return asRepositoryPredicateBuilderLike(builder).and(parts);
}

export function repositoryPredicateOr(builder: RepositoryPredicateBuilder, parts: RepositoryPredicate[]): RepositoryPredicate {
  return asRepositoryPredicateBuilderLike(builder).or(parts);
}
