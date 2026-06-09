// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// WebModelStore: unified query state, result snapshots, and base service signatures.
import type { ScopedStore } from '@/web/web/stores/storeScopeManager/types';
import type { QueryState } from '@/web/web/query/state';
import type { DataSetSnapshot } from '@/web/web/query/types';
import type { BaseModel } from '@/core/service';
import type { ClientModelService } from '@/core/rpc';

// Selection dropdown option.
export type SelectionItem = { value: string; label: string };

// Field metadata.
export type FieldMetadata = {
  id: string;
  type: string;
  typeAnnotation: string;
  notNull?: boolean;
  size?: number;
  precision?: number;
  scale?: number;
  scaleField?: string;
  round?: string;
  isReadonly?: boolean;
  indexed?: boolean;
  selection?: readonly SelectionItem[];
  relation?: string;
  relationModel?: string;
  relationFilter?: string;
  relationModelParentField?: string;
  relationInverseField?: string;
  relationJoinModel?: string;
  relationJoinField?: string;
  relationInverseJoinField?: string;
};

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
