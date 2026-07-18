<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="OSearchView"
    :show-header="showHeader"
    :action-ids="{ create: sessionActions.create, delete: sessionActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="UserId.Username" :label="_t('User')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="IpAddress" :label="_t('IP Address')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="Status" :label="_t('Status')" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ODateTimeField prop="LastActivityAt" :label="_t('Last Activity')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ODateTimeField prop="ExpiresAt" :label="_t('Expires At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ODateTimeField prop="CreatedAt" :label="_t('Created At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Session from '@/auth/service/models/session';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'SessionListView', inheritAttrs: true });
const { _t } = createTranslate('auth', { scope: 'web/views/SessionListView' });
const { _t: _tRef } = createTranslate('auth', { output: 'reference', scope: 'web/views/SessionListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store: WebModelStore<Session>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const { store, showHeader } = props;
const sessionActions = defineModelActions('auth.Session', { entityTitle: _tRef('Session') });
const { hasAction } = usePermission();

/**
 * Open the clicked session record in detail view.
 */
function onRowClick(payload: RowEventPayload<Session>) {
  router.push(`/auth/sessions/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<Session>();
defineExpose(expose);
</script>
