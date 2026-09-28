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
    :action-ids="{ create: roleActions.create, delete: roleActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="Code" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="Description" :store="store" :vColumnProps="{ minWidth: 200 }" />
    <ChoyBooleanField :store="store" prop="IsActive" widget="checkbox" />
    <ChoyBooleanField :store="store" prop="IsSystem" widget="checkbox" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Role from '@/auth/service/models/role';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyBooleanField, ChoyDatetimeField, ChoyListView, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleListView' });


const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Role>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleListView');
const { showHeader } = props;
const roleActions = defineModelActions('auth.Role', { entityTitle: _lt('Role') });
const { hasAction } = usePermission();

/**
 * Open the clicked role record in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  router.push(`/auth/roles/${(row as any).Id}`);
}

const { listRef, expose } = useListViewExpose<Role>();
defineExpose(expose);
</script>
