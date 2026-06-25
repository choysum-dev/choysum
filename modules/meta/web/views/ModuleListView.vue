<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="OSearchView"
    :show-header="showHeader"
    :action-ids="{ delete: moduleIndexActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <template #header-right>
      <el-button-group>
        <el-tooltip v-if="canRoute('meta.route.module_board')" content="看板视图" placement="top">
          <el-button :icon="GridViewSharp" @click="toKanban" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_list')" content="列表视图" placement="top">
          <el-button :icon="FormatListBulletedOutlined" @click="toList" type="primary" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_history')" content="操作历史" placement="top">
          <el-button :icon="HistoryOutlined" @click="toHistory" />
        </el-tooltip>
        <el-tooltip v-if="hasAction(moduleSyncIndexAction)" content="同步索引" placement="top">
          <el-button :icon="Refresh" :loading="syncLoading" @click="onSyncIndex" />
        </el-tooltip>
      </el-button-group>
    </template>

    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="ModuleName" label="模块名" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <OVarCharField prop="LocalVersion" label="本地版本" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="RegistryVersion" label="仓库版本" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="Version" label="展示版本" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="InstalledStatus" label="安装状态" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="InstalledVersion" label="已装版本" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OBooleanField prop="Available" label="可用" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OVarCharField prop="OriginTypes" label="来源" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="LocalPath" label="路径" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <ODateTimeField prop="LastSyncAt" label="同步时间" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type IrModuleIndex from '@/meta/service/models/ir_module_index';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import type { RowEventPayload } from '@/web/web/components/view/OListView.vue';
import { ElButton, ElTooltip, ElButtonGroup, ElMessage } from 'element-plus';
import { FormatListBulletedOutlined, GridViewSharp, HistoryOutlined } from '@vicons/material';
import { Refresh } from '@element-plus/icons-vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineAction, defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'ModuleListView', inheritAttrs: true });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store: WebModelStore<IrModuleIndex>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const { store, showHeader } = props;
const moduleSyncIndexAction = defineAction('meta.action.module_sync_index', {
  title: '同步模块索引',
  requires: [{ model: 'meta.IrModuleIndex', method: 'RequestSync' }],
});
const moduleIndexActions = defineModelActions('meta.IrModuleIndex', {
  entityTitle: '模块索引',
  titles: {
    edit: '编辑模块索引',
    copy: '复制模块索引',
    delete: '删除模块索引',
  },
});
const { canRoute, hasAction } = usePermission();

const syncLoading = ref(false);
const autoSyncTriggered = ref(false);
async function onSyncIndex() {
  if (syncLoading.value) return;
  syncLoading.value = true;
  try {
    const jobId = await (store as any).RequestSync({ originType: 'local', force: true, ifStale: false });
    ElMessage.success(jobId ? `已触发同步任务：${jobId}` : '已触发同步任务');
  } catch (error: any) {
    ElMessage.error(error?.message || '触发同步失败');
  } finally {
    syncLoading.value = false;
  }
}

watch(
  () => (store as any)?.state?.result,
  result => {
    if (autoSyncTriggered.value) return;
    const rows = (result as any)?.rows as any[] | undefined;
    if (!Array.isArray(rows)) return;
    if (rows.length > 0) return;
    autoSyncTriggered.value = true;
    void onSyncIndex();
  }
);

function onRowClick(payload: RowEventPayload<IrModuleIndex>) {
  router.push(`/meta/modules/${payload.row.Id}`);
}

function toKanban() {
  router.push('/meta/modules');
}
function toList() {
  router.push('/meta/modules/list');
}
function toHistory() {
  router.push('/meta/modules/history');
}

const { listRef, expose } = useListViewExpose<IrModuleIndex>();
defineExpose(expose);
</script>
