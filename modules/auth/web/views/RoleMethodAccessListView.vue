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
    :action-ids="{ create: methodAccessActions.create, delete: methodAccessActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OManyToOneRefField prop="IrApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OManyToOneRefField prop="IrModelId" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <OManyToOneRefField prop="IrServiceId" :store="store" :vColumnProps="{ minWidth: 180 }" />
    <OSelectionField prop="Mode" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OSelectionField prop="Source" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ODateTimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleMethodAccess from '@/auth/service/models/role_method_access';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleMethodAccessListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleMethodAccessListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store: WebModelStore<RoleMethodAccess>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const { store, showHeader } = props;
const methodAccessActions = defineModelActions('auth.RoleMethodAccess', { entityTitle: _lt('Method Access') });
const { hasAction } = usePermission();

/**
 * Open the clicked method-access row in detail view.
 */
function onRowClick(payload: RowEventPayload<RoleMethodAccess>) {
  router.push(`/auth/method-accesses/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<RoleMethodAccess>();
defineExpose(expose);
</script>
