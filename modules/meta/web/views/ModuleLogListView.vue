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
    :action-ids="{ delete: moduleLogActions.delete }"
    :has-action="hasAction"
  >
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="ModuleName" label="模块" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="Action" label="动作" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OVarCharField prop="ResultStatus" label="结果" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="OperatorUserId" label="操作人" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ODateTimeField prop="JobCreatedAt" label="开始时间" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ODateTimeField prop="JobFinishedAt" label="结束时间" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <OVarCharField prop="ErrorDomain" label="错误域" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="ErrorCode" label="错误码" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OJsonobjectField prop="SummaryJson" label="摘要" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <OJsonobjectField prop="LastErrorJson" label="错误详情" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <OIntField prop="Attempt" label="尝试次数" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OIntField prop="MaxAttempts" label="最大尝试" :store="store" :vColumnProps="{ minWidth: 100 }" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type ModuleManagementLog from '@/meta/service/models/module_management_log';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OJsonobjectField from '@/web/web/components/field/OJsonobjectField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'ModuleLogListView', inheritAttrs: true });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<ModuleManagementLog>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const { store, showHeader } = props;
const moduleLogActions = defineModelActions('meta.ModuleManagementLog', {
  entityTitle: '模块操作历史',
  titles: {
    delete: '删除操作历史',
  },
});
const { hasAction } = usePermission();

const { listRef, expose } = useListViewExpose<ModuleManagementLog>();
defineExpose(expose);
</script>
