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
    :action-ids="{ create: fieldRuleActions.create, delete: fieldRuleActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OManyToOneRefField prop="MetaApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OManyToOneRefField prop="MetaModelId" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <OManyToOneRefField prop="MetaFieldId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OSelectionField prop="LogicalModelName" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OSelectionField prop="PermRead" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OSelectionField prop="PermWrite" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ODateTimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleFieldRule from '@/auth/service/models/role_field_rule';
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

defineOptions({ name: 'RoleFieldRuleListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleFieldRuleListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleFieldRule>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleFieldRuleListView');
const { showHeader } = props;
const fieldRuleActions = defineModelActions('auth.RoleFieldRule', { entityTitle: _lt('Field Rule') });
const { hasAction } = usePermission();

/**
 * Open the clicked field-rule row in detail view.
 */
function onRowClick(payload: RowEventPayload<RoleFieldRule>) {
  router.push(`/auth/field-rules/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<RoleFieldRule>();
defineExpose(expose);
</script>
