<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Admin ledger for web.UserFilter (Settings label: Saved Filters).
    End-user Favorites create/apply stays on OSearch; this List is sys.admin
    cross-model cleanup and technical CRUD.
  -->
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    create-action="/web/user-filters/new"
    :action-ids="{
      create: userFilterActions.create,
      delete: userFilterActions.delete,
    }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end', width: 44 }" />
    <ChoyVarcharField
      :store="store"
      prop="Name"
      :label="_t('Name')"
      :vColumnProps="{ minWidth: 160 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="Application"
      :label="_t('Application')"
      :vColumnProps="{ minWidth: 120 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="ModelName"
      :label="_t('Model Name')"
      :vColumnProps="{ minWidth: 140 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="ScopeKey"
      :label="_t('Scope Key')"
      :vColumnProps="{ minWidth: 160 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="UserId"
      :label="_t('User')"
      :vColumnProps="{ minWidth: 120 }"
    />
    <ChoyBooleanField
      :store="store"
      prop="IsDefault"
      :label="_t('Is Default')"
      :vColumnProps="{ minWidth: 100 }"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  type RowEventPayload,
} from '@/web';
import { userFilterActions } from './user_filter_actions';

defineOptions({ name: 'UserFilterListView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/UserFilterListView' });

const router = useRouter();

const props = defineProps<{
  store?: WebModelStore<any>;
}>();

const store = resolvePageStore(props.store, 'UserFilterListView');
const { hasAction } = usePermission();

function onRowClick(payload: RowEventPayload) {
  const id = String(payload.row?.Id ?? '').trim();
  if (id) router.push(`/web/user-filters/${id}`);
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
