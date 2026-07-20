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
          <OVarCharField :store="store" prop="ModuleName" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="Version" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="InstalledStatus" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="InstalledVersion" />
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OBooleanField :store="store" prop="Available" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="OriginType" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="OriginRef" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="LocalPath" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="mdd-card">
      <template #header>
        <div class="mdd-card__header"><span>{{ _t('Sync Information') }}</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="LastSyncAt" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="LastBatchSyncAt" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="SyncRevision" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OTextField :store="store" prop="LastErrorMessage" />
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
          <ODateTimeField :store="store" prop="CreatedAt" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="UpdatedAt" />
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

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleDetailView' });

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
  entityTitle: _lt('Module Index'),
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
