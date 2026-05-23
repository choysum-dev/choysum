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
        <div class="mdd-card__header"><span>基本信息</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="ModuleName" label="模块名" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="Version" label="版本" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="InstalledStatus" label="安装状态" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="InstalledVersion" label="已装版本" />
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OBooleanField :store="store" prop="Available" label="可用" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="OriginType" label="来源类型" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="OriginRef" label="来源标识" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="LocalPath" label="本地路径" />
        </el-col>
      </el-row>
    </el-card>

    <el-card shadow="never" class="mdd-card">
      <template #header>
        <div class="mdd-card__header"><span>同步信息</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="LastSyncAt" label="最近同步" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="LastBatchSyncAt" label="批次同步" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OVarCharField :store="store" prop="SyncRevision" label="同步版本" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <OTextField :store="store" prop="LastErrorMessage" label="错误信息" />
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
        <div class="mdd-card__header"><span>时间信息</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="CreatedAt" label="创建时间" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6">
          <ODateTimeField :store="store" prop="UpdatedAt" label="更新时间" />
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

defineOptions({ name: 'ModuleDetailView', inheritAttrs: true });

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
  entityTitle: '模块索引',
  titles: {
    edit: '编辑模块索引',
    copy: '复制模块索引',
    delete: '删除模块索引',
  },
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
