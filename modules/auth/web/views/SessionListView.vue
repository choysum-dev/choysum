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
    :action-ids="{ create: sessionActions.create, delete: sessionActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="UserId.Username" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyVarcharField prop="IpAddress" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="Status" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyDatetimeField prop="LastActivityAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyDatetimeField prop="ExpiresAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Session from '@/auth/service/models/session';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyDatetimeField, ChoyListView, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';

defineOptions({ name: 'SessionListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/SessionListView' });


const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Session>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'SessionListView');
const { showHeader } = props;
const sessionActions = defineModelActions('auth.Session', { entityTitle: _lt('Session') });
const { hasAction } = usePermission();

/**
 * Open the clicked session record in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/auth/sessions/${id}`);
}

const { listRef, expose } = useListViewExpose<Session>();
defineExpose(expose);
</script>
