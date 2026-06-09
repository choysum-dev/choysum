// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { ComparisonOperatorExpression } from 'kysely';
import type BaseModel from '../../model/model';
import type { FilteredQueryProperties, Selectable } from './common';
import type { FieldSelection } from './selection';
import type { NonNil } from './shared';

/**
 * Comparison operators supported by repository query conditions.
 */
export type Operator = ComparisonOperatorExpression | 'contains' | 'child_of' | 'parent_of';

/**
 * Primitive tuple condition used by repository query expressions.
 */
export type BaseCondition = readonly [field: string, op: Operator, value: unknown];

/**
 * Conjunction node for base query conditions.
 */
export type BaseConditionAnd = { And: Array<BaseQueryCondition> };

/**
 * Disjunction node for base query conditions.
 */
export type BaseConditionOr = { Or: Array<BaseQueryCondition> };

/**
 * Logical node used by base query conditions.
 */
export type BaseQueryConditionNode = BaseConditionAnd | BaseConditionOr;

/**
 * Untyped repository condition tree.
 */
export type BaseQueryCondition = BaseCondition | BaseQueryConditionNode;

type Depth = [never, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9];

type NestedFilteredProp<T, Path, D extends number = 3> = D extends 0
  ? never
  : Path extends string
    ? Path extends keyof FilteredQueryProperties<T>
      ? T[Path]
      : Path extends `${infer K}.${infer Rest}`
        ? K extends keyof FilteredQueryProperties<T>
          ? T[K] extends BaseModel
            ? NestedFilteredProp<T[K], Rest, Depth[D]>
            : never
          : never
        : never
    : never;

/**
 * Nested field paths that remain queryable after property filtering.
 */
export type NestedPath<T, D extends number = 3> = D extends 0
  ? never
  : {
      [K in keyof FilteredQueryProperties<T>]: T[K] extends BaseModel ? K | `${K & string}.${NestedPath<T[K], Depth[D]> & string}` : K;
    }[keyof FilteredQueryProperties<T>];

type ConditionPathValue<T, K extends NestedPath<T, D>, D extends number = 3> =
  NonNil<NestedFilteredProp<T, K, D>> extends BaseModel
    ? string
    : NonNil<NestedFilteredProp<T, K, D>> extends Array<infer U>
      ? U extends BaseModel
        ? string
        : NestedFilteredProp<T, K, D>
      : NestedFilteredProp<T, K, D>;

type ConditionScalar<V> = V | null | undefined;
type ConditionValue<V> = ConditionScalar<V> | Array<ConditionScalar<V>>;

type Condition<T, D extends number = 3> = {
  [K in NestedPath<T, D>]: [field: K, op: Operator, value: ConditionValue<ConditionPathValue<T, K, D>>];
}[NestedPath<T, D>];

type SingleCondition<T> = Condition<Selectable<T>, 3>;
type QueryConditionNode<T> = { And: Array<QueryCondition<T>> } | { Or: Array<QueryCondition<T>> };

/**
 * Typed repository condition tree.
 */
export type QueryCondition<T> = SingleCondition<T> | QueryConditionNode<T>;

/**
 * Sort specification for repository reads.
 */
export type OrderBy<T> = {
  field: Extract<keyof T, string>;
  order: 'asc' | 'desc';
};

/**
 * Soft-delete visibility mode.
 */
export type SoftDeleteMode = 'default' | 'withDeleted' | 'onlyDeleted';

/**
 * Options that control soft-delete visibility.
 */
export interface SoftDeleteOptions {
  withDeleted?: boolean;
  onlyDeleted?: boolean;
}

/**
 * Search options accepted by repository reads.
 */
export interface SearchOptions<T> extends SoftDeleteOptions {
  fields?: FieldSelection<T>;
  limit?: number;
  offset?: number;
  orderBy?: OrderBy<T> | OrderBy<T>[];
  forUpdate?: boolean;
}

/**
 * Count options accepted by repository counts.
 */
export interface CountOptions extends SoftDeleteOptions {}

/**
 * Update options accepted by repository writes.
 */
export interface UpdateOptions extends SoftDeleteOptions {}

/**
 * Delete options accepted by repository deletes.
 */
export interface DeleteOptions extends SoftDeleteOptions {}
