<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader }"
    :action-ids="{ edit: moduleCatalogActions.edit, copy: moduleCatalogActions.copy, delete: moduleCatalogActions.delete }"
    :has-action="hasAction"
  >
    <ChoyCard :title="_t('Basic Information')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="ModuleName" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="LocalVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="RegistryVersion" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="InstalledVersion" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="Installed" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="Available" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="UpdateAvailable" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="LocalPath" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Sync Information')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyDatetimeField :store="store" prop="LastSyncAt" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="ActiveOperationId" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="Application" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyTextField :store="store" prop="UnavailableReason" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Dependencies')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="DependsJson" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyJsonField :store="store" prop="DependentsJson" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Timestamps')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyDatetimeField :store="store" prop="CreatedAt" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyDatetimeField :store="store" prop="UpdatedAt" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleCatalog from '@/meta/service/models/module_catalog';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyCard,
  ChoyCol,
  ChoyDatetimeField,
  ChoyFormView,
  ChoyGrid,
  ChoyJsonField,
  ChoyTextField,
  ChoyVarcharField,
} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'ModuleDetailView', inheritAttrs: true });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleDetailView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleCatalog>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);

const store = resolvePageStore(props.store, 'ModuleDetailView');
const { recordId, viewMode, showHeader } = props;
const moduleCatalogActions = defineModelActions('meta.MetaModuleCatalog', {
  entityTitle: _lt('Module Catalog'),
});
const { hasAction } = usePermission();
</script>
