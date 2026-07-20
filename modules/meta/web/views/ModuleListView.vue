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
        <el-tooltip v-if="canRoute('meta.route.module_board')" :content="_t('Board View')" placement="top">
          <el-button :icon="GridViewSharp" @click="toKanban" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_list')" :content="_t('List View')" placement="top">
          <el-button :icon="FormatListBulletedOutlined" @click="toList" type="primary" />
        </el-tooltip>
        <el-tooltip v-if="canRoute('meta.route.module_history')" :content="_t('Operation History')" placement="top">
          <el-button :icon="HistoryOutlined" @click="toHistory" />
        </el-tooltip>
        <el-tooltip v-if="hasAction(moduleSyncIndexAction)" :content="_t('Sync Index')" placement="top">
          <el-button :icon="Refresh" :loading="syncLoading" @click="onSyncIndex" />
        </el-tooltip>
      </el-button-group>
    </template>

    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="ModuleName" :label="_t('Module Name')" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <OVarCharField prop="LocalVersion" :label="_t('Local Version')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="RegistryVersion" :label="_t('Registry Version')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="Version" :label="_t('Display Version')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="InstalledStatus" :label="_t('Install Status')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="InstalledVersion" :label="_t('Installed Version')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OBooleanField prop="Available" :label="_t('Available')" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OVarCharField prop="OriginTypes" :label="_t('Origin')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="LocalPath" :label="_t('Path')" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <ODateTimeField prop="LastSyncAt" :label="_t('Synced At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
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
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import { ElButton, ElTooltip, ElButtonGroup, ElMessage } from 'element-plus';
import { FormatListBulletedOutlined, GridViewSharp, HistoryOutlined } from '@vicons/material';
import { Refresh } from '@element-plus/icons-vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineAction, defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ModuleListView', inheritAttrs: true });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleListView' });

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
  title: _lt('Sync Module Index'),
  requires: [{ model: 'meta.IrModuleIndex', method: 'RequestSync' }],
});
const moduleIndexActions = defineModelActions('meta.IrModuleIndex', {
  entityTitle: _lt('Module Index'),
});
const { canRoute, hasAction } = usePermission();

const syncLoading = ref(false);
const autoSyncTriggered = ref(false);
async function onSyncIndex() {
  if (syncLoading.value) return;
  syncLoading.value = true;
  try {
    const jobId = await (store as any).RequestSync({ force: true, ifStale: false });
    ElMessage.success(jobId ? _t('Sync job triggered: all:%s', String(jobId)) : _t('Sync job triggered'));
  } catch (error: any) {
    ElMessage.warning(_t('Sync failed: %s', String(error?.message || 'request failed')));
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
