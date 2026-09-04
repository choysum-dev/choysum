// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Build QueryContext objects from store state and override inputs
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { PaginationState, OrderByState } from './state';
import type { GroupBySpec } from './types';
import { filtersToQuery } from '@/web/web/query/utils/condition/builder';
import { combinePresentConditions } from '@/web/web/query/utils/condition/absent';

export interface QueryContext {
  shape: 'collection' | 'groups'; // query execution shape
  model: string; // logical model name (e.g., 'Task')
  // Minimal unified state for planner; derived from store.state.queryState and overrides
  queryState: { appliedGroups?: GroupBySpec<any>[]; orderBy?: OrderByState[]; pagination: PaginationState };
  recordId?: string; // when present, planner will build a 'browse' plan but snapshot still 'collection'
  // Optional shared filters
  filters?: any;
  // Range cursor info (for virtualization / partial fetch)
  range?: { offset: number; limit: number };
  // Execution options influencing plan construction (chart/full hierarchy etc.)
  options?: { fullGroupHierarchy?: boolean; skipPagination?: boolean; skipCount?: boolean };
}

export interface BuildContextOverrides {
  // Deprecated: model is ignored. Model name is strictly derived from store.storeId.
  model?: string;
  shape?: 'collection' | 'groups';
  queryState?: Partial<{ appliedGroups?: GroupBySpec<any>[]; orderBy?: OrderByState[]; pagination: PaginationState }>;
  recordId?: string;
  filters?: any;
  range?: { offset: number; limit: number };
  options?: { fullGroupHierarchy?: boolean; skipPagination?: boolean; skipCount?: boolean };
}

// Derive model name strictly from store.storeId
export function modelNameFromStoreId(storeId: string): string {
  if (!storeId) {
    throw new Error('modelNameFromStoreId requires a non-empty store.storeId');
  }
  const seg = String(storeId).split(':').pop() as string; // expected format like "namespace:Model" or "Model"
  return seg.charAt(0).toUpperCase() + seg.slice(1);
}

export function buildSearchOrGroupContext(store: WebModelStore<any>, overrides: BuildContextOverrides = {}): QueryContext {
  // Lock model to store.storeId only
  const baseModel = modelNameFromStoreId(store.storeId);
  const wantedShape = overrides.shape;
  const hasGroupby = !!((overrides.queryState as any)?.appliedGroups?.length || (store.state as any)?.queryState?.appliedGroups?.length);

  const shape: 'collection' | 'groups' = wantedShape ?? (hasGroupby ? 'groups' : 'collection');

  if (shape === 'collection') {
    const qs: any = (store.state as any)?.queryState || {};
    const base = { orderBy: qs.orderBy as OrderByState[] | undefined, pagination: (qs.pagination as PaginationState) ?? { limit: 100, offset: 0 } };
    const merged = {
      orderBy: overrides.queryState?.orderBy ?? base.orderBy,
      pagination: { ...base.pagination, ...(overrides.queryState?.pagination || {}) },
    } as { orderBy?: OrderByState[]; pagination: PaginationState };
    return {
      shape,
      model: baseModel,
      queryState: merged,
      // Executable filters are supplied through overrides instead of store state.
      filters: overrides.filters,
      range: overrides.range ?? { offset: merged.pagination.offset, limit: merged.pagination.limit },
      recordId: overrides.recordId,
      options: overrides.options,
    };
  }

  const qs: any = (store.state as any)?.queryState || {};
  const baseG = {
    appliedGroups: (qs.appliedGroups as GroupBySpec<any>[]) ?? [],
    orderBy: (qs.orderBy as OrderByState[] | undefined) ?? undefined,
    pagination: (qs.pagination as PaginationState) ?? { limit: 100, offset: 0 },
  };
  const mergedG = {
    appliedGroups: (overrides.queryState?.appliedGroups as GroupBySpec<any>[] | undefined) ?? baseG.appliedGroups,
    orderBy: overrides.queryState?.orderBy ?? baseG.orderBy,
    pagination: { ...baseG.pagination, ...(overrides.queryState?.pagination || {}) },
  } as { appliedGroups?: GroupBySpec<any>[]; orderBy?: OrderByState[]; pagination: PaginationState };
  return {
    shape,
    model: baseModel,
    queryState: mergedG,
    // Executable filters are supplied through overrides instead of store state.
    filters: overrides.filters,
    range: overrides.range ?? { offset: mergedG.pagination.offset, limit: mergedG.pagination.limit },
    options: overrides.options,
  };
}

export function buildBrowseContext(store: WebModelStore<any>, recordId: string, overrides: BuildContextOverrides = {}): QueryContext {
  // Lock model to store.storeId only
  const baseModel = modelNameFromStoreId(store.storeId);
  return {
    shape: 'collection',
    model: baseModel,
    queryState: {
      pagination: { limit: 100, offset: 0 },
      orderBy: undefined,
    },
    recordId,
    filters: overrides.filters,
  };
}

// Unified query builder that combines UI filters, keywords, forcedCondition, and parent conditions.
export function buildUnifiedQuery(
  store: WebModelStore<any>,
  options: {
    groupby?: GroupBySpec<any>[];
    pagination?: PaginationState;
    orderBy?: OrderByState[];
    parentCondition?: any;
    execOptions?: { fullGroupHierarchy?: boolean; skipPagination?: boolean; skipCount?: boolean };
  } = {}
): QueryContext {
  const qs: any = (store.state as any)?.queryState || {};
  const ui = Array.isArray(qs.appliedFilters) ? qs.appliedFilters : [];
  const kw = qs.keyword as string | undefined;
  const kwFields = qs.keywordFields as string[] | undefined;
  const uiCondition = filtersToQuery(ui, kw, kwFields, (store as any)?.fieldsMetadata);

  // Merge forced conditions with any supplied parent condition.
  const mergedFilters = combinePresentConditions(combinePresentConditions(uiCondition, qs.forcedCondition), options.parentCondition);

  const hasGroups = Array.isArray(options.groupby) && options.groupby.length > 0;
  const overrides: BuildContextOverrides = hasGroups
    ? {
        shape: 'groups',
        filters: mergedFilters,
        queryState: { appliedGroups: options.groupby as any, orderBy: options.orderBy, pagination: options.pagination as any },
        options: options.execOptions,
      }
    : {
        shape: 'collection',
        filters: mergedFilters,
        queryState: { orderBy: options.orderBy, pagination: options.pagination as any },
        options: options.execOptions,
      };

  return buildSearchOrGroupContext(store, overrides);
}
