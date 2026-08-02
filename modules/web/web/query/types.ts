// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// Unified dataset snapshot and row models for web controllers

// ========= Shared search/grouping UI and query helper types =========
// These replace previous imports from '@/core/rpc/search'
import type { QueryCondition, TemporalGranularity, GroupBySpec } from '@/core/service/api/query';
export type { GroupBySpec, TemporalGranularity };

// UI filter atom
export interface Condition {
  id: string;
  field: string;
  operator: string;
  value: any;
}

// UI filter group (can be nested). Optional name is used for preset labeling.
export interface ConditionGroup {
  id: string;
  // Prefer 'And' | 'Or' for consistency; legacy 'AND'|'OR' should be normalized upstream.
  logic: 'And' | 'Or';
  children: Array<Condition | ConditionGroup>;
  name?: string; // Optional preset label for this group
}

// Named filter preset. If name is missing, it's considered a forced filter (always applied, hidden from UI tags)
export type NamedFilter<T = any> = {
  name: string; // preset display name (required)
  query: QueryCondition<T>;
  // UI hint: whether this preset should be selected/applied initially
  selected?: boolean;
};

// Grouping preset with required name for consistent labeling
export type NamedGrouping<T = any> = {
  groupby: GroupBySpec<T> | Array<GroupBySpec<T>>;
};

// Query result snapshot kind: 'search' for flat records, 'group' for aggregated groups
export type QueryKind = 'search' | 'group';

// Explicit row kind for discriminating record vs group rows
export type RowKind = 'record' | 'group';

export interface AttachmentFieldDescriptor {
  kind: 'attachment';
  fieldType: 'binary' | 'image';
  fieldName: string;
  attachmentBindingId: string;
  ownerModel?: string;
  ownerRecordId?: string;
  fileName?: string;
  displayName?: string;
  previewUrl?: string;
}

export interface RecordRow {
  kind: 'record';
  key: string; // Stable key, usually record id
  payload: any; // Backend record payload

  // Group-local row index starting from 1 when expanded inside grouped results.
  groupIndex?: number;
}

export interface GroupRow {
  kind: 'group';
  key: string; // Stable grouped key produced by the backend.
  depth: number; // Group nesting depth, where top level is 0.
  label: string; // Primary display label.
  count?: number; // Record count within the group.
  metrics?: Record<string, any>; // Aggregate alias to aggregate value.
  __condition?: any; // Backend condition used for child-group or detail fetching.
  keys?: Record<string, any>; // Alias to raw grouped key values.
  labels?: Record<string, string>; // Alias to formatted display labels.
  children?: GroupRow[]; // Optional recursively nested child groups.
  raw?: any; // Full backend row kept for debugging or advanced views.
}

export interface DataSetSnapshot {
  kind: QueryKind; // 'search' (Search/Browse) | 'group' (ReadGroup)
  rows: (RecordRow | GroupRow)[];
  total?: number; // collection: records count; groups: groups count
  planHash?: string; // Signature of the main plan
  ts: number; // Last update timestamp
  error?: any;
  uiView?: string; // Optional: current UI view (list/kanban/form)
  groupRecords?: Map<string, RecordRow[]>; // Optional: cache for expanded group records
  meta?: Record<string, any>; // Extension slot for future metadata
}

// =============================
// QueryDriver related shared types (centralized)
// =============================

import type { PaginationState, OrderByState } from '@/web/web/query/state';
import type { ComputedRef, Ref } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { BaseModel } from '@/core/rpc';

// Note: legacy QueryDriver-related types and APIs have been removed.

export type ForcedFilter<T = any> = { query: QueryCondition<T> };
export interface HasPagination {
  pagination: PaginationState;
}

// =============================
// Unified controller/view model types (centralization of previous controllerInterfaces & local defs)
// =============================

// Base controller view model (list / kanban / form share loading + error + result snapshot)
export interface BaseControllerVM {
  loading: boolean;
  error: unknown | null;
  result: DataSetSnapshot | null;
}

// List view model extends base with visible flattened nodes & expanded group keys
export interface ListViewModel extends BaseControllerVM {
  visibleNodes: (GroupRow | RecordRow)[];
  expandedKeys: Set<string>;
}

// Form view mode & model
export type FormMode = 'display' | 'edit' | 'create';
export interface FormViewModel extends BaseControllerVM {
  mode: FormMode;
  draft: any | null;
  original: any | null;
}

// Kanban lane abstraction (first-level group column)
export type Lane = {
  key: string;
  label: string;
  count?: number;
  condition?: any;
  raw?: any;
  [k: string]: any;
};

// Centralized operator option used by filter UI and helpers
export interface OperatorOption {
  value: string;
  label: string;
}

// QueryUpdatePayload mirrors QueryState for UI-emitted search updates.
export interface QueryUpdatePayload<T = any> {
  keyword?: string;
  appliedFilters: ConditionGroup[]; // always an array (empty => explicit clear)
  appliedGroups?: Array<GroupBySpec<T>>; // undefined -> no change / none; empty [] -> explicit clear
}

// SearchViewComponent constrains the searchView prop without relying on DefineComponent invariance.
export type SearchViewComponent<T extends BaseModel = any> = new (...args: any[]) => {
  $props: { store: WebModelStore<T> } & Record<string, any>;
  $emit: (e: 'query-update', payload: QueryUpdatePayload<T>) => void;
};

// Browse (list-like) controller public interface
export interface IBrowseViewController {
  vm: ListViewModel; // list/kanban use the list VM shape
  apply(overrides?: {
    forcedCondition?: QueryCondition<any>; // external forced condition (merged with user filters)
    appliedFilters?: ConditionGroup[]; // UI ConditionGroup[] tree (tags)
    keyword?: string;
    keywordFields?: string[];
    orderBy?: OrderByState[];
    pagination?: PaginationState;
    appliedGroups?: Array<GroupBySpec<any>>;
    kind?: QueryKind;
  }): Promise<DataSetSnapshot | void>;
  paginate(p: PaginationState): Promise<void>;
  sort(orderBy: OrderByState[]): Promise<void>;
  setForcedCondition(cond: QueryCondition<any> | QueryCondition<any>[]): Promise<void>;
  setAppliedFilters(filters: ConditionGroup[]): Promise<void>;
  setKeyword(keyword: string | undefined): Promise<void>;
  setKeywordFields(keywordFields: string[]): Promise<void>;
  setAppliedGroups(groupby: Array<GroupBySpec<any>>): Promise<void>;
}

// List controller extends browse with group expansion helpers
export interface IListController extends IBrowseViewController {
  expandGroup(key: string, expanded: boolean): Promise<void>;
  fetchGroupChildren(key: string): Promise<GroupRow[]>;
  fetchGroupRecords(key: string, pagination?: PaginationState, options?: { skipCount?: boolean }): Promise<RecordRow[] | null>;
  loadMoreGroupRecords(key: string): Promise<void>;
  loadMoreGroupChildren(key: string): Promise<void>;
  bootstrapDefaults?(defaultFilter?: any, defaultGroups?: any): void;
}

// Kanban controller interface (wraps underlying list controller operations + lane helpers)
export interface IKanbanController {
  vm: IListController['vm'];
  lanes: ComputedRef<Lane[]>;
  laneRecords: Ref<Record<string, RecordRow[]>>;
  apply: (overrides?: {
    forcedCondition?: QueryCondition<any>; // external forced condition (merged with user filters)
    appliedFilters?: ConditionGroup[]; // ConditionGroup[] tree for UI tags
    keyword?: string;
    keywordFields?: string[];
    orderBy?: OrderByState[];
    pagination?: PaginationState;
    appliedGroups?: Array<GroupBySpec<any>>;
    kind?: QueryKind;
  }) => Promise<any>;
  paginate: (p: PaginationState) => Promise<void>;
  setAppliedGroups: (gb: Array<GroupBySpec<any>>) => Promise<void>;
  setKeyword: (keyword: string | undefined) => Promise<void>;
  setKeywordFields: (keywordFields: string[]) => Promise<void>;
  setForcedCondition: (cond: QueryCondition<any> | QueryCondition<any>[]) => Promise<void>;
  setAppliedFilters: (filters: ConditionGroup[]) => Promise<void>;
  sort: (orderBy: OrderByState[]) => Promise<void>;
  getLaneField: () => string | null;
  getLaneValue: (lane: Lane) => any;
  getLaneRemain: (lane: Lane) => number;
  preloadLane: (laneKey: string) => Promise<void>;
  loadMoreLane: (laneKey: string) => Promise<void>;
  moveCard: (cardId: string, fromLaneKey: string, toLaneKey: string, index: number) => Promise<void>;
  refreshLaneAggregates: (keys: string[]) => Promise<void>;
}

// Form controller interface
export interface IFormViewController {
  vm: FormViewModel;
  beginDisplay(recordId: string): Promise<any>;
  beginCreate(initial?: any): Promise<void>;
  beginEdit(): void;
  reset(): void;
  validate(): Promise<{ valid: boolean; errors: any[] }>;
  submit(): Promise<any>;
  delete(): Promise<any>;
  provideToChildren(key?: string): { setField(path: string, value: any): void; getField(path: string): any; onChange(cb: (draft: any) => void): void };
}
