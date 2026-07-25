// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

/* eslint-disable */
import { defineStore } from 'pinia';
import { computed, reactive } from 'vue';
import type {{.Model.Name}} from '{{ConvertPathNoExt .Model.Path}}';
import type { IStoreScopeManager } from '@/web/web/stores/storeScopeManager/types';
import type { WebModelStore, PlanCacheEntry } from '@/web/web/stores/modelStore';
import { createFieldsGetHelpers } from '@/web/web/stores/fieldsGet';
import { createEmptyQueryState } from '@/web/web/query/state';
import type { DataSetSnapshot } from '@/web/web/query/types';
import { clearFieldsByStore } from '@/web/web/query/utils/registry/field';
import { clearMetricsByStore } from '@/web/web/query/utils/registry/metric';
import { registerStoreFactory, createStoreByModel } from '@/web/web/stores/registry';
import { webApiService } from '@/core/web/client';
import { {{.Model.Name}}Service } from '../service';
import { useI18nStore } from '@/web/web/stores/i18nStore';
/**
 * API client for the {{.Model.Name}} model.
 */
export const {{.Model.Name}}Api = webApiService('{{.App}}.{{.Model.Name}}', {{.Model.Name}}Service);

// Field metadata.
export const {{.Model.Name}}FieldsMetadata = {
{{- range $field := .FieldsMetadata}}
  {{$field.Name}}: {
    {{- if $field.Id}}id: '{{$field.Id}}',{{- end}}
    type: '{{$field.FieldType}}',
    typeAnnotation: '{{$field.TypeAnnotation}}',
    {{- if $field.StorageKind}}storageKind: '{{$field.StorageKind}}',{{- end}}
    {{- if ne $field.ShouldCreateColumn nil}}shouldCreateColumn: {{$field.ShouldCreateColumn}},{{- end}}
    {{- if $field.ResolvedColumnType}}resolvedColumnType: '{{$field.ResolvedColumnType}}',{{- end}}
    {{- if $field.ReasonCode}}reasonCode: '{{$field.ReasonCode}}',{{- end}}
    {{- if $field.ComputedKind}}computedKind: '{{$field.ComputedKind}}',{{- end}}
    {{- if $field.RelatedPath}}relatedPath: '{{$field.RelatedPath}}',{{- end}}
    {{- if ne $field.RelatedStore nil}}relatedStore: {{$field.RelatedStore}},{{- end}}
    {{- if ne $field.Searchable nil}}searchable: {{$field.Searchable}},{{- end}}
    {{- if and $field.RelationModel (or (eq $field.FieldType "ManyToOne") (eq $field.FieldType "OneToMany") (eq $field.FieldType "ManyToMany") (eq $field.FieldType "ManyToOneRef") (eq $field.FieldType "ManyToManyRef"))}}relationModel: '{{$field.RelationModel}}',{{- end}}
    {{- if $field.RelationFilter}}relationFilter: "{{$field.RelationFilter}}",{{- end}}
    {{- if $field.RelationModelParentField}}relationModelParentField: '{{$field.RelationModelParentField}}',{{- end}}
    {{- if $field.NotNull}}notNull: {{$field.NotNull}},{{- end}}
    {{- if $field.Size}}size: {{$field.Size}},{{- end}}
    {{- if $field.Precision}}precision: {{$field.Precision}},{{- end}}
    {{- if $field.Scale}}scale: {{$field.Scale}},{{- end}}
    {{- if $field.ScaleField}}scaleField: '{{$field.ScaleField}}',{{- end}}
    {{- if $field.Round}}round: '{{$field.Round}}',{{- end}}
    {{- if $field.IsReadonly}}isReadonly: {{$field.IsReadonly}},{{- end}}
    {{- if $field.Indexed}}indexed: {{$field.Indexed}},{{- end}}
    {{- if $field.Translate}}translate: {{$field.Translate}},{{- end}}
    {{- if $field.RelationInverseField}}relationInverseField: '{{$field.RelationInverseField}}',{{- end}}
    {{- if $field.RelationJoinModel}}relationJoinModel: '{{$field.RelationJoinModel}}',{{- end}}
    {{- if $field.RelationJoinField}}relationJoinField: '{{$field.RelationJoinField}}',{{- end}}
    {{- if $field.RelationInverseJoinField}}relationInverseJoinField: '{{$field.RelationInverseJoinField}}',{{- end}}
    {{- if $field.String}}string: {{$field.String}},{{- end}}
    {{- if $field.StringText}}stringText: {{$field.StringText}},{{- end}}
    {{- if $field.SelectionKind}}selectionKind: '{{$field.SelectionKind}}',{{- end}}
    {{- if $field.Selection}}selection: {{$field.Selection}},{{- end}}
  },
{{- end}}
} as const;

// Do not redeclare base service methods from WebModelStore; only append custom services.
export interface {{.Model.Name}}Store extends WebModelStore<{{.Model.Name}}> {
  {{- range $service := .Model.Services}}
  {{- if not (IsBaseService $service.Name) }}
  {{$service.Name}}: (...args: Parameters<typeof {{$.Model.Name}}Api.{{$service.Name}}>) => ReturnType<typeof {{$.Model.Name}}Api.{{$service.Name}}>;
  {{- end}}
  {{- end}}
}

const {{.Model.Name | ToLowerCase}}StoreRegistry = new Map<string, {{.Model.Name}}Store>();

export type Create{{.Model.Name}}StoreOptions = {
  // Optional custom storeId used to distinguish scoped instances; it must be non-empty.
  storeId?: string;
  scopeManager?: IStoreScopeManager;
};

export function create{{.Model.Name}}Store(options?: Create{{.Model.Name}}StoreOptions): {{.Model.Name}}Store {
  const { scopeManager } = options || {};
  // Convention: storeId must stay stable and non-empty; default to "{{.Model.Name}}" without timestamps.
  const storeId = options?.storeId && options.storeId.trim() ? options.storeId.trim() : `{{.Model.Name}}`;

  if ({{.Model.Name | ToLowerCase}}StoreRegistry.has(storeId)) {
    return {{.Model.Name | ToLowerCase}}StoreRegistry.get(storeId)!;
  }

  const store = defineStore(storeId, () => {
    const state = reactive({
      queryState: createEmptyQueryState(),
      result: undefined as DataSetSnapshot | undefined,
      selection: [] as string[],
      planCache: new Map<string, PlanCacheEntry>(),
    });

    const isLoading = computed(() => {{.Model.Name}}Api.isLoading.value);
    const lastError = computed(() => {{.Model.Name}}Api.lastError.value);

    function resetState() {
      state.queryState = createEmptyQueryState(state.queryState.kind);
      state.result = undefined;
      state.selection = [];
      state.planCache.clear();
    }

    const fieldsGetHelpers = createFieldsGetHelpers(
      {
        fieldsMetadata: {{.Model.Name}}FieldsMetadata as any,
        FieldsGet: {{.Model.Name}}Api.FieldsGet as any,
      },
      {
        getLang: () => {
          try {
            return String(useI18nStore().terminologyLang || '').trim() || 'en_US';
          } catch {
            return 'en_US';
          }
        },
      }
    );

    function destroy() {
      (store as any)._destroyed = true;
      fieldsGetHelpers.clearFieldsGetCache();
      {{.Model.Name | ToLowerCase}}StoreRegistry.delete(storeId);
      clearFieldsByStore(storeId);
      clearMetricsByStore(storeId);
      resetState();
    }

    return {
      storeId,
      fullModelName: '{{.App}}.{{.Model.Name}}',
      application: '{{.App}}',
      modelName: '{{.Model.Name}}',
      state,
      get isLoading() { return isLoading.value; },
      get lastError() { return lastError.value; },
      destroy,
      fieldsMetadata: {{.Model.Name}}FieldsMetadata,
      ensureFieldsGet: fieldsGetHelpers.ensureFieldsGet,
      getFieldMeta: fieldsGetHelpers.getFieldMeta,
      getFieldsGetTranslatedString: fieldsGetHelpers.getFieldsGetTranslatedString,
      clearFieldsGetCache: fieldsGetHelpers.clearFieldsGetCache,
      setContext: {{.Model.Name}}Api.setContext,
      getContext: {{.Model.Name}}Api.getContext,
      withContext: {{.Model.Name}}Api.withContext,
      {{- range $service := .Model.Services}}
      {{$service.Name}}: {{$.Model.Name}}Api.{{$service.Name}},
      {{- end}}
    } as {{.Model.Name}}Store;
  })();

  {{.Model.Name | ToLowerCase}}StoreRegistry.set(storeId, store);
  if (scopeManager) scopeManager.register(store);
  return store;
}

export function use{{.Model.Name}}Store(storeId: string): {{.Model.Name}}Store | undefined {
  return {{.Model.Name | ToLowerCase}}StoreRegistry.get(storeId);
}

export function useGlobal{{.Model.Name}}Store(): {{.Model.Name}}Store {
  // Global singleton: fixed storeId "{{.Model.Name}}.Global" still resolves the model name from the suffix.
  const globalStoreId = '{{.Model.Name}}.Global';
  if (!{{.Model.Name | ToLowerCase}}StoreRegistry.has(globalStoreId)) {
    return create{{.Model.Name}}Store({ storeId: globalStoreId });
  }
  return {{.Model.Name | ToLowerCase}}StoreRegistry.get(globalStoreId)!;
}

registerStoreFactory('{{.App}}.{{.Model.Name}}', create{{.Model.Name}}Store);
