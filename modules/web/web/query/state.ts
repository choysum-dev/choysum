// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

import type { NamedFilter, ConditionGroup, GroupBySpec, NamedGrouping, QueryKind } from '@/web/web/query/types';
import type { QueryCondition } from '@/core/service/api/query';

/**
 * Pagination state stored alongside a query.
 */
export interface PaginationState {
  /** Requested page size. */
  limit: number;

  /** Zero-based row offset. */
  offset: number;
}

/**
 * Sort order definition stored alongside a query.
 */
export interface OrderByState {
  /** Field used for sorting. */
  field: string;

  /** Sort direction. */
  direction: 'asc' | 'desc';
}

/**
 * Unified query state used by the web model store.
 */
export interface QueryState<T = any> {
  /** Query mode such as search or grouped reading. */
  kind: QueryKind;

  /** Forced condition merged with user conditions during execution. */
  forcedCondition?: QueryCondition<T>;

  /**
   * When set, Search/Count And the source field's meta `condition`
   * (same as NameSearch `relationConditionSource`).
   */
  relationConditionSource?: { model: string; field: string };

  /** Applied UI filter groups. */
  appliedFilters: ConditionGroup[];

  /** Default named filters injected on first use. */
  defaultFilters?: NamedFilter<T>[];

  /** Keyword retained in state and translated into query conditions later. */
  keyword?: string;

  /** Fields searched by the keyword query. */
  keywordFields?: string[];

  /** Active grouping definitions. */
  appliedGroups: GroupBySpec<T>[];

  /** Default groupings injected when no grouping has been applied yet. */
  defaultGroups?: NamedGrouping<T>[];

  /** Pagination state. */
  pagination: PaginationState;

  /** Optional order-by definitions. */
  orderBy?: OrderByState[];
}

/**
 * Creates an empty query state with default pagination.
 */
export function createEmptyQueryState<T = any>(kind: QueryKind = 'search'): QueryState<T> {
  return {
    kind,
    forcedCondition: undefined,
    relationConditionSource: undefined,
    appliedFilters: [],
    defaultFilters: [],
    keyword: undefined,
    keywordFields: undefined,
    appliedGroups: [],
    defaultGroups: [],
    pagination: { limit: 100, offset: 0 },
    orderBy: undefined,
  };
}

type RelationPickerQueryStore = {
  state?: { queryState?: QueryState | Record<string, unknown> };
};

/**
 * Clear leftover user search UI on a relation picker's store before Search more opens.
 * Drops keyword / applied filters & groups / page offset and clear-default flags.
 * Keeps defaultFilters / defaultGroups / orderBy / keywordFields; List props re-apply
 * forcedCondition and relationConditionSource on mount.
 */
export function resetRelationPickerQueryState(store: RelationPickerQueryStore | null | undefined): void {
  const state = store?.state;
  if (!state || typeof state !== 'object') return;
  const qs = (state.queryState && typeof state.queryState === 'object' ? state.queryState : {}) as Record<
    string,
    unknown
  >;
  const prevPag = (qs.pagination && typeof qs.pagination === 'object' ? qs.pagination : {}) as PaginationState;
  const limit = Number(prevPag.limit);
  state.queryState = {
    ...qs,
    kind: 'search',
    keyword: undefined,
    appliedFilters: [],
    appliedGroups: [],
    forcedCondition: undefined,
    relationConditionSource: undefined,
    userClearedDefaultFilters: undefined,
    userClearedDefaultGroups: undefined,
    pagination: {
      limit: Number.isFinite(limit) && limit > 0 ? limit : 100,
      offset: 0,
    },
  };
}
