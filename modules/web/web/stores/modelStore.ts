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
  DefaultGet: ClientModelService<typeof BaseModel.DefaultGet<TModel>>;
  Create: ClientModelService<typeof BaseModel.Create<TModel>>;
  CreateMany: ClientModelService<typeof BaseModel.CreateMany<TModel>>;
  Browse: ClientModelService<typeof BaseModel.Browse<TModel>>;
  BrowseMany: ClientModelService<typeof BaseModel.BrowseMany<TModel>>;
  Update: ClientModelService<typeof BaseModel.Update<TModel>>;
  UpdateById: ClientModelService<typeof BaseModel.UpdateById<TModel>>;
  Copy: ClientModelService<typeof BaseModel.Copy<TModel>>;
  NameSearch: ClientModelService<typeof BaseModel.NameSearch<TModel>>;
  NameCreate: ClientModelService<typeof BaseModel.NameCreate<TModel>>;
  Count: ClientModelService<typeof BaseModel.Count<TModel>>;
  Search: ClientModelService<typeof BaseModel.Search<TModel>>;
  ReadGroup: ClientModelService<typeof BaseModel.ReadGroup<TModel>>;
  ReadGroupCount: ClientModelService<typeof BaseModel.ReadGroupCount<TModel>>;
  Delete: ClientModelService<typeof BaseModel.Delete<TModel>>;
  DeleteById: ClientModelService<typeof BaseModel.DeleteById<TModel>>;
  Onchange: ClientModelService<typeof BaseModel.Onchange<TModel>>;
  FieldsGet: ClientModelService<typeof BaseModel.FieldsGet>;
  GetFieldTranslations: ClientModelService<typeof BaseModel.GetFieldTranslations>;
  UpdateFieldTranslations: ClientModelService<typeof BaseModel.UpdateFieldTranslations>;
  GetFieldCompanyValues: ClientModelService<typeof BaseModel.GetFieldCompanyValues>;
  UpdateFieldCompanyValues: ClientModelService<typeof BaseModel.UpdateFieldCompanyValues>;

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
