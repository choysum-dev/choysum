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
    :action-ids="{ create: uiResourceGrantActions.create, delete: uiResourceGrantActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OSelectionField prop="Mode" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OManyToOneRefField prop="MetaApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OManyToOneRefField prop="MetaUiResourceId" :store="store" :vColumnProps="{ minWidth: 200 }" />
    <ODateTimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleUiResource from '@/auth/service/models/role_ui_resource';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleUiResourceListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleUiResourceListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleUiResource>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleUiResourceListView');
const { showHeader } = props;
const uiResourceGrantActions = defineModelActions('auth.RoleUiResource', { entityTitle: _lt('UI Resource Grant') });
const { hasAction } = usePermission();

/**
 * Open the clicked UI-resource grant row in detail view.
 */
function onRowClick(payload: RowEventPayload<RoleUiResource>) {
  router.push(`/auth/ui-resource-grants/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<RoleUiResource>();
defineExpose(expose);
</script>
