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
 * Untyped leaf tuple condition used by repository query expressions.
 */
export type UntypedCondition = readonly [field: string, op: Operator, value: unknown];

/**
 * Conjunction node for untyped query conditions.
 */
export type UntypedConditionAnd = { And: Array<UntypedQueryCondition> };

/**
 * Disjunction node for untyped query conditions.
 */
export type UntypedConditionOr = { Or: Array<UntypedQueryCondition> };

/**
 * Logical node used by untyped query conditions.
 */
export type UntypedQueryConditionNode = UntypedConditionAnd | UntypedConditionOr;

/**
 * Untyped repository condition tree (dynamic / authz / engine).
 */
export type UntypedQueryCondition = UntypedCondition | UntypedQueryConditionNode;

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
 *
 * `-?` is required: mapping over optional keys without it makes
 * `{ [K in keyof T]: K }[keyof T]` include `undefined`, which collapses
 * {@link QueryCondition} so that arbitrary strings (and typos) assign.
 */
export type NestedPath<T, D extends number = 3> = D extends 0
  ? never
  : {
      [K in keyof FilteredQueryProperties<T>]-?: T[K] extends BaseModel ? K | `${K & string}.${NestedPath<T[K], Depth[D]> & string}` : K;
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

/**
 * Typed leaf condition for one model.
 */
export type QueryConditionLeaf<T> = Condition<Selectable<T>, 3>;

/**
 * Typed And/Or node for one model.
 */
export type QueryConditionNode<T> = { And: Array<QueryCondition<T>> } | { Or: Array<QueryCondition<T>> };

/**
 * Typed repository condition tree.
 */
export type QueryCondition<T> = QueryConditionLeaf<T> | QueryConditionNode<T>;

/**
 * Assert an untyped condition tree as {@link QueryCondition} for one model.
 * Used when And/Or trees are built dynamically and cannot be inferred as typed fields.
 *
 * Prefer a bare literal when it already assigns to the Search/Count parameter
 * (statically known field names on a sound {@link QueryCondition}).
 * Use this helper for dynamic trees (variable field names, `any[]` Or parts).
 * Pass an explicit type argument for method-generic `Model.Search<T>` when nesting
 * would otherwise poison `T`.
 */
export function condition<T>(tree: UntypedQueryCondition): QueryCondition<T> {
  return tree as QueryCondition<T>;
}

/**
 * Sort specification for repository reads.
 */
export type OrderBy<T> = {
  field: Extract<keyof Selectable<T>, string>;
  order: 'asc' | 'desc';
};

/**
 * Options that control soft-delete visibility.
 */
export interface SoftDeleteOptions {
  withDeleted?: boolean;
  onlyDeleted?: boolean;
}

/**
 * Source relational field pointer for candidate Search / NameSearch.
 * BE resolves `@Field({ condition })` on that field and Ands it into the query.
 */
export type RelationConditionSource = {
  /** Source model technical name (`@Model` / MetadataStorage key). */
  model: string;
  /** Source relation field property name on that model. */
  field: string;
};

/**
 * Search options accepted by repository reads.
 */
export interface SearchOptions<T> extends SoftDeleteOptions {
  fields?: FieldSelection<T>;
  limit?: number;
  offset?: number;
  orderBy?: OrderBy<T> | OrderBy<T>[];
  forUpdate?: boolean;
  /**
   * When set, BE Ands the source field's meta `condition` (static or callable)
   * into this search. Used by relational typeahead / Search-more.
   */
  relationConditionSource?: RelationConditionSource;
}
