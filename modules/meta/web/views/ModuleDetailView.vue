<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader }"
    :action-ids="{ edit: moduleIndexActions.edit, copy: moduleIndexActions.copy, delete: moduleIndexActions.delete }"
    :has-action="hasAction"
  >
    <ChoyCard :title="_t('Basic Information')" class="mdd-card"><ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="ModuleName" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="Version" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="InstalledStatus" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="InstalledVersion" />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyBooleanField :store="store" prop="Available" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="OriginType" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="OriginRef" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="LocalPath" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Sync Information')" class="mdd-card"><ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyDatetimeField :store="store" prop="LastSyncAt" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyDatetimeField :store="store" prop="LastBatchSyncAt" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="SyncRevision" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyTextField :store="store" prop="LastErrorMessage" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard title="Manifest" class="mdd-card"><ChoyGrid :cols="12">
        <ChoyCol :span="12">
          <ChoyJsonField :store="store" prop="ManifestJson" label="ManifestJson" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>

    <ChoyCard :title="_t('Timestamps')" class="mdd-card"><ChoyGrid :cols="12">
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
import type MetaModuleIndex from '@/meta/service/models/module_index';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyJsonField, ChoyTextField, ChoyVarcharField } from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'ModuleDetailView', inheritAttrs: true });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleDetailView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleIndex>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);

const store = resolvePageStore(props.store, 'ModuleDetailView');
const { recordId, viewMode, showHeader } = props;
const moduleIndexActions = defineModelActions('meta.MetaModuleIndex', {
  entityTitle: _lt('Module Index'),
});
const { hasAction } = usePermission();
</script>

<style scoped>
.mdd-card {
  margin-bottom: 14px;
}
</style>
