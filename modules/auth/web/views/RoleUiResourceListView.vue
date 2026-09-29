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
    :action-ids="{ create: uiResourceGrantActions.create, delete: uiResourceGrantActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoySelectionField prop="Mode" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyManyToOneRefField prop="MetaApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyManyToOneRefField prop="MetaUiResourceId" :store="store" :vColumnProps="{ minWidth: 200 }" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleUiResource from '@/auth/service/models/role_ui_resource';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoySelectionField, ChoyVColumn, ChoyVarcharField, ChoyManyToOneRefField} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';

defineOptions({ name: 'RoleUiResourceListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleUiResourceListView' });


const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleUiResource>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleUiResourceListView');
const { showHeader } = props;
const uiResourceGrantActions = defineModelActions('auth.RoleUiResource', { entityTitle: _lt('UI Resource Grant') });
const { hasAction } = usePermission();

/**
 * Open the clicked UI-resource grant row in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/auth/ui-resource-grants/${id}`);
}

const { listRef, expose } = useListViewExpose<RoleUiResource>();
defineExpose(expose);
</script>
