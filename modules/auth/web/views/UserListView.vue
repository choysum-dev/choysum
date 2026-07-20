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
    :action-ids="{ create: userActions.create, delete: userActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />

    <OImageField prop="Avatar" :label="_t('Avatar')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="Username" :label="_t('Username')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OManyToOneRefField :store="store" prop="CompanyId" :label="_t('Company')" />
    <OVarCharField prop="Email" :label="_t('Email')" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <OVarCharField prop="Phone" :label="_t('Phone')" :store="store" />
    <OVarCharField prop="FullName" :label="_t('Full Name')" :store="store" />
    <ODateTimeField prop="CreatedAt" :label="_t('Created At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type User from '@/auth/service/models/user';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OImageField from '@/web/web/components/field/OImageField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

const router = useRouter();

defineOptions({ name: 'UserListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/UserListView' });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<User>;
    showHeader?: boolean;
  }>(),
  { showHeader: true }
);

const { store, showHeader } = props;
const userActions = defineModelActions('auth.User', { entityTitle: _lt('User') });
const { hasAction } = usePermission();

/**
 * Open the clicked user record in detail view.
 */
function onRowClick(payload: RowEventPayload<User>) {
  router.push(`/auth/users/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<User>();
defineExpose(expose);
</script>
