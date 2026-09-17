// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: Apache-2.0

// WebModelStore: unified query state, result snapshots, and base service signatures.
import type { ScopedStore } from '@/web/web/stores/storeScopeManager/types';
import type { QueryState } from '@/web/web/query/state';
import type { DataSetSnapshot } from '@/web/web/query/types';
import type { BaseModel } from '@/core/service';
import type { ClientModel, ClientModelService, FieldSelection, Insertable, Updateable } from '@/core/rpc';
import type {
  CountOptions,
  DeleteOptions,
  QueryCondition,
  ReadGroupCountOptions,
  ReadGroupOptions,
  SearchOptions,
  SoftDeleteOptions,
  UpdateOptions,
} from '@/core/service/api/query';
import type { Projected, RowOrProjected } from '@/core/service/api/selection';
import type { OnchangeDraft, OnchangeResult } from '@/core/service/runtime/onchange/types';
import type { OnchangeTrigger } from '@/core/service/orm/metadata/field';
import type { ResolvePropertiesOptions } from '@/core/service/orm/model/properties_resolve';
import type { ResolvedPropertyItem } from '@/core/service/orm/model/properties_types';
import type { CopyOptions } from '@/core/service/orm/model/model_copy';
import type { NameCreateOptions } from '@/core/service/orm/model/model_namecreate';
import type { GroupBySpec, ReadGroupResult } from '@/core/service/orm/repository/types/groupby';
import type { TermReference } from '@/core/service/i18n';

/**
 * Row-bound BaseModel method shapes for the FE store.
 * Field-selecting CRUD methods use call-site generics so literal `fields`
 * preserve {@link Projected} / {@link RowOrProjected} (wrapping via ClientModelService
 * freezes Parameters/ReturnType).
 */
type StoreDefaultGet<T extends BaseModel> = (value: Partial<Insertable<T>>) => Promise<Partial<Insertable<T>>>;
type StoreCreate<T extends BaseModel> = <F extends FieldSelection<T> | undefined = undefined>(
  value: Partial<Insertable<T>>,
  returnFields?: F
) => Promise<ClientModel<RowOrProjected<T, F>>>;
type StoreCreateMany<T extends BaseModel> = <F extends FieldSelection<T> | undefined = undefined>(
  values: Partial<Insertable<T>>[],
  returnFields?: F
) => Promise<Array<ClientModel<RowOrProjected<T, F>>>>;
type StoreBrowse<T extends BaseModel> = <F extends FieldSelection<T> | undefined = undefined>(
  id: string,
  fields?: F,
  options?: SoftDeleteOptions
) => Promise<ClientModel<RowOrProjected<T, F>>>;
type StoreBrowseMany<T extends BaseModel> = <F extends FieldSelection<T> | undefined = undefined>(
  ids: string[],
  fields?: F,
  options?: SoftDeleteOptions
) => Promise<Array<ClientModel<RowOrProjected<T, F>>>>;
type StoreUpdate<T extends BaseModel> = {
  <F extends FieldSelection<T>>(
    condition: QueryCondition<T>,
    values: Partial<Updateable<T>>,
    returnFields: F,
    options?: UpdateOptions
  ): Promise<Array<ClientModel<Projected<T, F>>>>;
  (
    condition: QueryCondition<T>,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<Array<ClientModel<Partial<T>>>>;
};
type StoreUpdateById<T extends BaseModel> = {
  <F extends FieldSelection<T>>(
    id: string,
    values: Partial<Updateable<T>>,
    returnFields: F,
    options?: UpdateOptions
  ): Promise<ClientModel<Projected<T, F>>>;
  (
    id: string,
    values: Partial<Updateable<T>>,
    returnFields?: FieldSelection<T>,
    options?: UpdateOptions
  ): Promise<ClientModel<Partial<T>>>;
};
type StoreCopy<T extends BaseModel> = (
  id: string,
  defaults?: Partial<Record<string, unknown>>,
  options?: CopyOptions
) => Promise<ClientModel<T>>;
type StoreNameSearch<T extends BaseModel> = <F extends FieldSelection<T> | undefined = undefined>(
  name: string,
  condition?: QueryCondition<T> | [],
  options?: Omit<SearchOptions<T>, 'fields'> & { fields?: F }
) => Promise<Array<ClientModel<RowOrProjected<T, F>>>>;
type StoreNameCreate<T extends BaseModel> = <F extends FieldSelection<T> | undefined = undefined>(
  name: string,
  values?: Partial<Insertable<T>>,
  options?: Omit<NameCreateOptions<T>, 'returnFields'> & { returnFields?: F }
) => Promise<ClientModel<RowOrProjected<T, F>>>;
type StoreCount<T extends BaseModel> = (condition?: QueryCondition<T> | [], options?: CountOptions) => Promise<number>;
type StoreSearch<T extends BaseModel> = <F extends FieldSelection<T> | undefined = undefined>(
  condition?: QueryCondition<T> | [],
  options?: Omit<SearchOptions<T>, 'fields'> & { fields?: F }
) => Promise<Array<ClientModel<RowOrProjected<T, F>>>>;
type StoreReadGroup<T extends BaseModel> = (
  groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
  condition?: QueryCondition<T> | [],
  options?: ReadGroupOptions<T>
) => Promise<ReadGroupResult>;
type StoreReadGroupCount<T extends BaseModel> = (
  groupby: Array<GroupBySpec<T> | GroupBySpec<T>[]> | [],
  condition?: QueryCondition<T> | [],
  options?: ReadGroupCountOptions<T>
) => Promise<number>;
type StoreDelete<T extends BaseModel> = (condition: QueryCondition<T>, options?: DeleteOptions) => Promise<number>;
type StoreDeleteById<T extends BaseModel> = (id: string, options?: DeleteOptions) => Promise<number>;
type StoreOnchange<T extends BaseModel> = (
  draft: OnchangeDraft,
  changed: OnchangeTrigger<T>[],
  opts?: { withCompute?: boolean; maxIterations?: number; loopThreshold?: number }
) => Promise<OnchangeResult<T>>;
type StoreResolveProperties<T extends BaseModel> = (
  record: Partial<T> | Record<string, unknown> | null | undefined,
  fieldName: string,
  opts?: ResolvePropertiesOptions
) => Promise<ResolvedPropertyItem[]>;

// Selection dropdown option.
export type SelectionItem = { value: string; label: string };

/**
 * Web / client field metadata (codegen static table + FieldsGet overlay).
 * Distinct from ORM `FieldMetadata` in core.
 */
export type WebFieldMetadata = {
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
  notNull?: boolean;
  size?: number;
  precision?: number;
  scale?: number;
  scaleField?: string;
  currencyField?: string;
  round?: string;
  isReadonly?: boolean;
  indexed?: boolean;
  /** Field title msgid (English) or FieldsGet-translated title. */
  string?: string;
  /** Field title TermReference for Gateway / translateTerm. */
  stringText?: TermReference;
  /** Field help msgid (English) or FieldsGet-translated help. */
  help?: string;
  /** Field help TermReference for Gateway / translateTerm. */
  helpText?: TermReference;
  selection?: readonly SelectionItem[];
  /** Present for dynamic selection fields (P3); static may omit or be 'static'. */
  selectionKind?: 'static' | 'dynamic';
  /** Data-i18n: field values stored as lang maps (see data-i18n-design). */
  translate?: boolean;
  /** Company-dependent: field values stored as company maps (see company-dependent-design). */
  companyDependent?: boolean;
  /** Per-field upload byte cap (image/binary; PR-P2-F3). */
  maxUploadBytes?: number;
  /** Pixel width cap (image only; PR-P2-F3). */
  maxWidth?: number;
  /** Pixel height cap (image only; PR-P2-F3). */
  maxHeight?: number;
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

export function getFieldMetadataView(meta: WebFieldMetadata | undefined) {
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
  readonly fieldsMetadata: Record<string, WebFieldMetadata>;

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

  // Base ORM RPCs shared by generated XxxStore (Go IsBaseService filters these names).
  // When adding a BaseModel public static async PascalCase method for the FE base surface,
  // declare it here; webapistore resolves names from BaseModel meta — do not maintain a
  // parallel name list in webapistore.go.
  DefaultGet: ClientModelService<StoreDefaultGet<TModel>>;
  Create: StoreCreate<TModel>;
  CreateMany: StoreCreateMany<TModel>;
  Browse: StoreBrowse<TModel>;
  BrowseMany: StoreBrowseMany<TModel>;
  Update: StoreUpdate<TModel>;
  UpdateById: StoreUpdateById<TModel>;
  Copy: ClientModelService<StoreCopy<TModel>>;
  NameSearch: StoreNameSearch<TModel>;
  NameCreate: StoreNameCreate<TModel>;
  Count: ClientModelService<StoreCount<TModel>>;
  Search: StoreSearch<TModel>;
  ReadGroup: ClientModelService<StoreReadGroup<TModel>>;
  ReadGroupCount: ClientModelService<StoreReadGroupCount<TModel>>;
  Delete: ClientModelService<StoreDelete<TModel>>;
  DeleteById: ClientModelService<StoreDeleteById<TModel>>;
  Onchange: ClientModelService<StoreOnchange<TModel>>;
  FieldsGet: ClientModelService<typeof BaseModel.FieldsGet>;
  GetFieldTranslations: ClientModelService<typeof BaseModel.GetFieldTranslations>;
  UpdateFieldTranslations: ClientModelService<typeof BaseModel.UpdateFieldTranslations>;
  GetFieldCompanyValues: ClientModelService<typeof BaseModel.GetFieldCompanyValues>;
  UpdateFieldCompanyValues: ClientModelService<typeof BaseModel.UpdateFieldCompanyValues>;
  ResolveProperties: ClientModelService<StoreResolveProperties<TModel>>;

  /**
   * Fetch (or reuse cached) FieldsGet presentation overlay for the active terminology lang.
   */
  ensureFieldsGet: (fields?: string[], attributes?: string[]) => Promise<Record<string, WebFieldMetadata>>;
  /** Merge static fieldsMetadata with FieldsGet overlay for one field. */
  getFieldMeta: (name: string) => WebFieldMetadata | undefined;
  /** FieldsGet-translated title when overlay is present for the active lang. */
  getFieldsGetTranslatedString: (name: string) => string | undefined;
  /** FieldsGet-translated help when overlay is present for the active lang. */
  getFieldsGetTranslatedHelp: (name: string) => string | undefined;
  clearFieldsGetCache: () => void;
}
