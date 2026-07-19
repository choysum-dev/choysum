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
    <OVarCharField prop="ModuleName" :label="_t('Module')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="Action" :label="_t('Action')" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OVarCharField prop="ResultStatus" :label="_t('Result')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="OperatorUserId" :label="_t('Operator')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ODateTimeField prop="JobCreatedAt" :label="_t('Started At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ODateTimeField prop="JobFinishedAt" :label="_t('Finished At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <OVarCharField prop="ErrorDomain" :label="_t('Error Domain')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="ErrorCode" :label="_t('Error Code')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OJsonobjectField prop="SummaryJson" :label="_t('Summary')" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <OJsonobjectField prop="LastErrorJson" :label="_t('Error Detail')" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <OIntField prop="Attempt" :label="_t('Attempts')" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OIntField prop="MaxAttempts" :label="_t('Max Attempts')" :store="store" :vColumnProps="{ minWidth: 100 }" />
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
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ModuleLogListView', inheritAttrs: true });

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleLogListView' });
const { _t: _tRef } = createTranslate('meta', { output: 'reference', scope: 'web/views/ModuleLogListView' });

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
  entityTitle: _tRef('Module Operation History'),
});
const { hasAction } = usePermission();

const { listRef, expose } = useListViewExpose<ModuleManagementLog>();
defineExpose(expose);
</script>
