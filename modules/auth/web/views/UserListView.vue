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
    :action-ids="{ create: userActions.create, delete: userActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />

    <ChoyImageField prop="Avatar" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="Username" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyManyToOneField :store="store" prop="CompanyId" />
    <ChoyVarcharField prop="Email" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <ChoyVarcharField prop="Phone" :store="store" />
    <ChoyVarcharField prop="FullName" :store="store" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type User from '@/auth/service/models/user/user';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyDatetimeField, ChoyImageField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';
import { createTranslate } from '@/web/web/i18n';


const router = useRouter();

defineOptions({ name: 'UserListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/UserListView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<User>;
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);

const store = resolvePageStore(props.store, 'UserListView');
const { showHeader } = props;
const userActions = defineModelActions('auth.User', { entityTitle: _lt('User') });
const { hasAction } = usePermission();

/**
 * Open the clicked user record in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  router.push(`/auth/users/${(row as any).Id}`);
}

const { listRef, expose } = useListViewExpose<User>();
defineExpose(expose);
</script>
