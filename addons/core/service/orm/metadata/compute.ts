// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { Operator, BaseQueryCondition, ExpressionWrapper } from '../repository/types';
import type { DialectName } from '../repository/repository_dialect';
import type { ObjectRecord } from '../../../utils/types';

/**
 * Execution identity used when compute handlers run.
 */
export type ComputeRunAs = 'user' | 'sudo';

/**
 * Operator type accepted by compute search handlers.
 */
export type ComputeOperator = Operator;

/**
 * Search domain returned by compute search handlers.
 */
export type ComputeSearchDomain = BaseQueryCondition;

/**
 * Non-empty dependency tuple declared by a compute field.
 */
export type ComputeDeps<TDep extends string = string> = readonly [TDep, ...TDep[]];

/**
 * Context passed to a compute inverse handler.
 */
export interface ComputeInverseContext<TModel, TValue> {
  model: TModel;
  value: TValue;
  previousValue: TValue | undefined;
  runAs: ComputeRunAs;
}

/**
 * Handler signature used to inverse-write computed values.
 */
export type ComputeInverseHandler<TModel, TValue> = (ctx: ComputeInverseContext<TModel, TValue>) => ObjectRecord | Promise<ObjectRecord>;

/**
 * Context passed to a compute search handler.
 */
export interface ComputeSearchContext<TDep extends string = string> {
  field: TDep;
  op: ComputeOperator;
  value: unknown;
  dialect: DialectName;
  runAs: ComputeRunAs;
}

/**
 * Result returned by a compute search handler.
 */
export type ComputeSearchHandlerResult =
  | { domain: ComputeSearchDomain; sql?: never }
  | { sql: ExpressionWrapper<ObjectRecord, string, unknown>; domain?: never };

/**
 * Handler signature used to translate compute-field searches into domain or SQL fragments.
 */
export type ComputeSearchHandler<TDep extends string = string> = (
  ctx: ComputeSearchContext<TDep>
) => ComputeSearchHandlerResult | Promise<ComputeSearchHandlerResult>;
