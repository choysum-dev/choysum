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
    detail-path="/auth/sessions/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="UserId.Username" :store="store" />
    <ChoyVarcharField prop="IpAddress" :store="store" />
    <ChoyVarcharField prop="Status" :store="store" />
    <ChoyDatetimeField prop="LastActivityAt" mode="datetime" :store="store" />
    <ChoyDatetimeField prop="ExpiresAt" mode="datetime" :store="store" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type Session from '@/auth/service/models/session';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyDatetimeField, ChoyListView, ChoySearchView, ChoyTableColumn, ChoyVarcharField } from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'SessionListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/SessionListView' });



const props = withDefaults(
  defineProps<{
    store?: ModelStore<Session>;
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

const { listRef, expose } = useListViewExpose<Session>();
defineExpose(expose);
</script>
