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
    :action-ids="{ create: partnerActions.create, delete: partnerActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />

    <OVarCharField :store="store" prop="Name" label="名称" :vColumnProps="{ minWidth: 180 }" />
    <OVarCharField :store="store" prop="Code" label="编码" :vColumnProps="{ minWidth: 120 }" />
    <OManyToOneRefField :store="store" prop="CompanyId" label="所属公司" :vColumnProps="{ minWidth: 180 }" />
    <OIntField :store="store" prop="CustomerRank" label="客户等级" />
    <OIntField :store="store" prop="SupplierRank" label="供应商等级" />
    <OBooleanField :store="store" prop="IsActive" label="启用" />
    <ODateTimeField :store="store" prop="UpdatedAt" label="更新时间" mode="datetime" :vColumnProps="{ minWidth: 160 }" />
  </OListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Partner from '@/partner/service/models/partner';
import OListView from '@/web/web/components/view/OListView.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineAction, defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';

defineOptions({ name: 'PartnerListView', inheritAttrs: true });

/**
 * Props consumed by the partner list view.
 */
defineProps<{
  store: WebModelStore<Partner>;
}>();

const router = useRouter();

/**
 * Action descriptor used to open the partner detail page from the list.
 */
const partnerOpenDetailAction = defineAction('partner.action.partner_open_detail', {
  title: '打开伙伴详情',
  requires: [{ model: 'partner.Partner' }],
});
const partnerActions = defineModelActions('partner.Partner', { entityTitle: '伙伴' });
const { hasAction } = usePermission();

/**
 * Opens the clicked partner row when the actor has detail access.
 */
function onRowClick(payload: RowEventPayload<Partner>) {
  if (!hasAction(partnerOpenDetailAction)) {
    return;
  }
  router.push(`/partner/partners/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<Partner>();
defineExpose(expose);
</script>
