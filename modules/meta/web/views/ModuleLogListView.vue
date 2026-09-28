<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :show-header="showHeader"
    :action-ids="{ delete: moduleLogActions.delete }"
    :has-action="hasAction"
  >
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="ModuleName" :label="_t('Module')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="Action" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyVarcharField prop="ResultStatus" :label="_t('Result')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="OperatorUserId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyDatetimeField prop="JobCreatedAt" :label="_t('Started At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyDatetimeField prop="JobFinishedAt" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyVarcharField prop="ErrorDomain" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="ErrorCode" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyJsonField prop="SummaryJson" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <ChoyJsonField prop="LastErrorJson" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <ChoyNumberField prop="Attempt" mode="integer" :label="_t('Attempts')" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyNumberField prop="MaxAttempts" mode="integer" :store="store" :vColumnProps="{ minWidth: 100 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type ModuleManagementLog from '@/meta/service/models/module_management_log';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { ChoyDatetimeField, ChoyJsonField, ChoyListView, ChoyNumberField, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';

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
