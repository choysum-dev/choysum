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
    <OVarCharField prop="Action" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OVarCharField prop="ResultStatus" :label="_t('Result')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="OperatorUserId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ODateTimeField prop="JobCreatedAt" :label="_t('Started At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ODateTimeField prop="JobFinishedAt" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <OVarCharField prop="ErrorDomain" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="ErrorCode" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OJsonobjectField prop="SummaryJson" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <OJsonobjectField prop="LastErrorJson" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <OIntField prop="Attempt" :label="_t('Attempts')" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OIntField prop="MaxAttempts" :store="store" :vColumnProps="{ minWidth: 100 }" />
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
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ModuleLogListView', inheritAttrs: true });

const { _t, _lt } = createTranslate('meta', { scope: 'web/views/ModuleLogListView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<ModuleManagementLog>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'ModuleLogListView');
const { showHeader } = props;
const moduleLogActions = defineModelActions('meta.ModuleManagementLog', {
  entityTitle: _lt('Module Operation History'),
});
const { hasAction } = usePermission();

const { listRef, expose } = useListViewExpose<ModuleManagementLog>();
defineExpose(expose);
</script>
