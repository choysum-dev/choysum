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
    :action-ids="{ create: roleActions.create, delete: roleActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="Name" :label="_t('Name')" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OVarCharField prop="Code" :label="_t('Code')" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <OVarCharField prop="Description" :label="_t('Description')" :store="store" :vColumnProps="{ minWidth: 200 }" />
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" widget="checkbox" />
    <OBooleanField :store="store" prop="IsSystem" :label="_t('Built-in')" widget="checkbox" />
    <ODateTimeField prop="CreatedAt" :label="_t('Created At')" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Role from '@/auth/service/models/role';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('auth', { scope: 'web/views/RoleListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store: WebModelStore<Role>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const { store, showHeader } = props;
const roleActions = defineModelActions('auth.Role', { entityTitle: _lt('Role') });
const { hasAction } = usePermission();

/**
 * Open the clicked role record in detail view.
 */
function onRowClick(payload: RowEventPayload<Role>) {
  router.push(`/auth/roles/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<Role>();
defineExpose(expose);
</script>
