<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader }"
    :action-ids="{ edit: moduleIndexActions.edit, copy: moduleIndexActions.copy, delete: moduleIndexActions.delete }"
    :has-action="hasAction"
  >
    <el-card shadow="never" class="mdd-card">
      <template #header>
        <div class="mdd-card__header"><span>{{ _t('Basic Information') }}</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="ModuleName" :label="_t('Module Name')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="Version" :label="_t('Version')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="InstalledStatus" :label="_t('Install Status')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="InstalledVersion" :label="_t('Installed Version')" />
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OBooleanField :store="store" prop="Available" :label="_t('Available')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="OriginType" :label="_t('Origin Type')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="OriginRef" :label="_t('Origin Ref')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="LocalPath" :label="_t('Local Path')" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="mdd-card">
      <template #header>
        <div class="mdd-card__header"><span>{{ _t('Sync Information') }}</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="LastSyncAt" :label="_t('Last Synced At')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="LastBatchSyncAt" :label="_t('Batch Synced At')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="SyncRevision" :label="_t('Sync Revision')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OTextField :store="store" prop="LastErrorMessage" :label="_t('Error Message')" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="mdd-card">
      <template #header>
        <div class="mdd-card__header"><span>Manifest</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="24">
          <OJsonobjectField :store="store" prop="ManifestJson" label="ManifestJson" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="mdd-card">
      <template #header>
        <div class="mdd-card__header"><span>{{ _t('Timestamps') }}</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="CreatedAt" :label="_t('Created At')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="UpdatedAt" :label="_t('Updated At')" />
        </el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type IrModuleIndex from '@/meta/service/models/ir_module_index';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OTextField from '@/web/web/components/field/OTextField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ModuleDetailView', inheritAttrs: true });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleDetailView' });
const { _t: _tRef } = createTranslate('meta', { output: 'reference', scope: 'web/views/ModuleDetailView' });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<IrModuleIndex>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);

const { store, recordId, viewMode, showHeader } = props;
const moduleIndexActions = defineModelActions('meta.IrModuleIndex', {
  entityTitle: _tRef('Module Index'),
});
const { hasAction } = usePermission();
</script>

<style scoped>
.mdd-card {
  margin-bottom: 14px;
}
.mdd-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
