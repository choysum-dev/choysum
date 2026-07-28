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
    :action-ids="{ create: recordRuleActions.create, delete: recordRuleActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OSelectionField prop="Kind" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <OManyToOneRefField prop="IrApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <OManyToOneRefField prop="IrModelId" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <OBooleanField prop="PermRead" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <OBooleanField prop="PermWrite" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <OBooleanField prop="PermCreate" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <OBooleanField prop="PermDelete" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <ODateTimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';
import OListView from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleRecordRuleListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleRecordRuleListView' });

const router = useRouter();

const props = withDefaults(
  defineProps<{
    store: WebModelStore<RoleRecordRule>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const { store, showHeader } = props;
const recordRuleActions = defineModelActions('auth.RoleRecordRule', { entityTitle: _lt('Record Rule') });
const { hasAction } = usePermission();

/**
 * Open the clicked record-rule row in detail view.
 */
function onRowClick(payload: RowEventPayload<RoleRecordRule>) {
  router.push(`/auth/record-rules/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<RoleRecordRule>();
defineExpose(expose);
</script>
