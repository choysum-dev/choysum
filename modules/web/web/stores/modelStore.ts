// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// WebModelStore: unified query state, result snapshots, and base service signatures.
import type { ScopedStore } from '@/web/web/stores/storeScopeManager/types';
import type { QueryState } from '@/web/web/query/state';
import type { DataSetSnapshot } from '@/web/web/query/types';
import type { BaseModel } from '@/core/service';
import type { ClientModelService } from '@/core/rpc';
import type { TermReference } from '@/core/service/i18n';

// Selection dropdown option.
export type SelectionItem = { value: string; label: string; labelText?: TermReference };

// Field metadata.
export type FieldMetadata = {
  id: string;
  type: string;
  typeAnnotation: string;
  storageKind?: string;
  shouldCreateColumn?: boolean;
  resolvedColumnType?: string;
  reasonCode?: string;
  computedKind?: string;
  relatedPath?: string;
  relatedStore?: boolean;
  searchable?: boolean;
  runAs?: string;
  notNull?: boolean;
  size?: number;
  precision?: number;
  scale?: number;
  scaleField?: string;
  round?: string;
  isReadonly?: boolean;
  indexed?: boolean;
  selection?: readonly SelectionItem[];
  relationModel?: string;
  relationFilter?: string;
  relationModelParentField?: string;
  relationInverseField?: string;
  relationJoinModel?: string;
  relationJoinField?: string;
  relationInverseJoinField?: string;
};

const RELATION_FIELD_TYPES = new Set(['manytoone', 'onetomany', 'manytomany', 'manytooneref', 'manytomanyref']);

export function isRelationFieldType(type: string | undefined): boolean {
  return RELATION_FIELD_TYPES.has(String(type || '').toLowerCase());
}

export function getFieldMetadataView(meta: FieldMetadata | undefined) {
  const type = typeof meta?.type === 'string' ? meta.type : '';

  const relationModel = meta?.relationModel;
  const relatedPath = meta?.relatedPath;
  const computedKind = meta?.computedKind;
  const storageKind = meta?.storageKind;
  const searchable = typeof meta?.searchable === 'boolean' ? meta.searchable : undefined;
  const shouldCreateColumn = typeof meta?.shouldCreateColumn === 'boolean' ? meta.shouldCreateColumn : undefined;
  const resolvedColumnType = meta?.resolvedColumnType;
  const reasonCode = meta?.reasonCode;
  const relatedStore = typeof meta?.relatedStore === 'boolean' ? meta.relatedStore : undefined;
  const runAs = meta?.runAs;

  return {
    relationModel,
    relatedPath,
    relatedStore,
    computedKind,
    storageKind,
    shouldCreateColumn,
    resolvedColumnType,
    reasonCode,
    searchable,
    runAs,
    isRelation: isRelationFieldType(type),
  } as const;
}

export type PlanCacheEntry = {
  signature: string;
  kind: string; // 'search'|'count'|'readGroup'|'readGroupCount'|'browse'
  hit: number;
  lastUsed: number;
  createdAt: number;
};

export interface WebModelStore<TModel extends BaseModel> extends ScopedStore {
  readonly storeId: string;
  // Fully qualified model name, for example auth.Role.
  readonly fullModelName?: string;
  // Application name, for example auth.
  readonly application?: string;
  // Model name, for example Role.
  readonly modelName?: string;
  readonly fieldsMetadata: Record<string, FieldMetadata>;

  state: {
    // Unified query state.
    queryState: QueryState<TModel>;
    result?: DataSetSnapshot;
    selection: string[];
    planCache: Map<string, PlanCacheEntry>;
  };

  readonly isLoading?: boolean;
  readonly lastError?: any;

  destroy(): void;

  // Context helpers.
  setContext: (ctx: Record<string, string>) => void;
  getContext: () => Record<string, string>;
  withContext: <T>(ctx: Record<string, string>, fn: () => Promise<T>) => Promise<T>;

  // Base services, optionally populated by generated templates.
  DefaultGet: ClientModelService<typeof BaseModel.DefaultGet<TModel>>;
  Create: ClientModelService<typeof BaseModel.Create<TModel>>;
  CreateMany: ClientModelService<typeof BaseModel.CreateMany<TModel>>;
  Browse: ClientModelService<typeof BaseModel.Browse<TModel>>;
  BrowseMany: ClientModelService<typeof BaseModel.BrowseMany<TModel>>;
  Update: ClientModelService<typeof BaseModel.Update<TModel>>;
  UpdateById: ClientModelService<typeof BaseModel.UpdateById<TModel>>;
  Count: ClientModelService<typeof BaseModel.Count<TModel>>;
  Search: ClientModelService<typeof BaseModel.Search<TModel>>;
  ReadGroup: ClientModelService<typeof BaseModel.ReadGroup<TModel>>;
  ReadGroupCount: ClientModelService<typeof BaseModel.ReadGroupCount<TModel>>;
  Delete: ClientModelService<typeof BaseModel.Delete<TModel>>;
  DeleteById: ClientModelService<typeof BaseModel.DeleteById<TModel>>;
  Onchange: ClientModelService<typeof BaseModel.Onchange<TModel>>;
}
