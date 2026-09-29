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
    :action-ids="{ create: methodAccessActions.create, delete: methodAccessActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyManyToOneRefField prop="MetaApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyManyToOneRefField prop="MetaModelId" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyManyToOneRefField prop="MetaServiceId" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <ChoySelectionField prop="LogicalModelName" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoySelectionField prop="Mode" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoySelectionField prop="Source" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleMethodAccess from '@/auth/service/models/role_method_access';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoySelectionField, ChoyVColumn, ChoyVarcharField, ChoyManyToOneRefField} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';

defineOptions({ name: 'RoleMethodAccessListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleMethodAccessListView' });


const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleMethodAccess>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleMethodAccessListView');
const { showHeader } = props;
const methodAccessActions = defineModelActions('auth.RoleMethodAccess', { entityTitle: _lt('Method Access') });
const { hasAction } = usePermission();

/**
 * Open the clicked method-access row in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/auth/method-accesses/${id}`);
}

const { listRef, expose } = useListViewExpose<RoleMethodAccess>();
defineExpose(expose);
</script>
