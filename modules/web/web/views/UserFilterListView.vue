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
    detail-path="/web/user-filters/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end' }" />
    <ChoyVarcharField
      :store="store"
      prop="Name"
      :label="_t('Name')"
    />
    <ChoyVarcharField
      :store="store"
      prop="Application"
      :label="_t('Application')"
    />
    <ChoyVarcharField
      :store="store"
      prop="ModelName"
      :label="_t('Model Name')"
    />
    <ChoyVarcharField
      :store="store"
      prop="ScopeKey"
      :label="_t('Scope Key')"
    />
    <ChoyVarcharField
      :store="store"
      prop="UserId"
      :label="_t('User')"
    />
    <ChoyBooleanField
      :store="store"
      prop="IsDefault"
      :label="_t('Is Default')"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
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
} from '@/web';
import { userFilterActions } from './user_filter_actions';

defineOptions({ name: 'UserFilterListView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/UserFilterListView' });


const props = defineProps<{
  store?: ModelStore<any>;
}>();

const store = resolvePageStore(props.store, 'UserFilterListView');
const { hasAction } = usePermission();


const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
