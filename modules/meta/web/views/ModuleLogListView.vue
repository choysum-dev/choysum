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
    :action-ids="{}"
    :has-action="hasAction"
  >
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="ModuleName" :label="_t('Module')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="Action" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyVarcharField prop="Status" :label="_t('Status')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="Phase" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="OperatorUserId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyDatetimeField prop="StartedAt" :label="_t('Started At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyDatetimeField prop="FinishedAt" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyVarcharField prop="ErrorDomain" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="ErrorCode" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyJsonField prop="PlanJson" :store="store" :vColumnProps="{ minWidth: 220 }" />
    <ChoyVarcharField prop="JobId" :store="store" :vColumnProps="{ minWidth: 140 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type MetaModuleOperation from '@/meta/service/models/module_operation';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { ChoyDatetimeField, ChoyJsonField, ChoyListView, ChoySearchView, ChoyTableColumn, ChoyVarcharField } from '@/web';

defineOptions({ name: 'ModuleLogListView', inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<MetaModuleOperation>;
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);

const { _t } = createTranslate('meta', { scope: 'web/views/ModuleLogListView' });
const { hasAction } = usePermission();

const store = resolvePageStore(props.store, 'ModuleLogListView') as WebModelStore<MetaModuleOperation>;
const showHeader = props.showHeader;
const { listRef, expose } = useListViewExpose<MetaModuleOperation>();
defineExpose(expose);
</script>
