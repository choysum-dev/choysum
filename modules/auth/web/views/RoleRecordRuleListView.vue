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
    :action-ids="{ create: recordRuleActions.create, delete: recordRuleActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoySelectionField prop="Kind" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyManyToOneField prop="MetaApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyManyToOneField prop="MetaModelId" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyBooleanField prop="PermRead" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <ChoyBooleanField prop="PermWrite" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <ChoyBooleanField prop="PermCreate" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <ChoyBooleanField prop="PermDelete" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyBooleanField, ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoySelectionField, ChoyVColumn, ChoyVarcharField } from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleRecordRuleListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleRecordRuleListView' });


const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleRecordRule>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleRecordRuleListView');
const { showHeader } = props;
const recordRuleActions = defineModelActions('auth.RoleRecordRule', { entityTitle: _lt('Record Rule') });
const { hasAction } = usePermission();

/**
 * Open the clicked record-rule row in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  router.push(`/auth/record-rules/${(row as any).Id}`);
}

const { listRef, expose } = useListViewExpose<RoleRecordRule>();
defineExpose(expose);
</script>
