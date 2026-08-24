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

    <OVarCharField :store="store" prop="Name" :vColumnProps="{ minWidth: 180 }" />
    <OVarCharField :store="store" prop="Code" :vColumnProps="{ minWidth: 120 }" />
    <OManyToOneRefField :store="store" prop="CompanyId" :vColumnProps="{ minWidth: 180 }" />
    <OIntField :store="store" prop="CustomerRank" />
    <OIntField :store="store" prop="SupplierRank" />
    <OBooleanField :store="store" prop="IsActive" />
    <ODateTimeField :store="store" prop="UpdatedAt" mode="datetime" :vColumnProps="{ minWidth: 160 }" />
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
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'PartnerListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('partner', { scope: 'web/views/PartnerListView' });

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
  title: _lt('Open Partner Detail'),
  requires: [{ model: 'partner.Partner' }],
});
const partnerActions = defineModelActions('partner.Partner', { entityTitle: _lt('Partner') });
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

const { expose } = useListViewExpose<Partner>();
defineExpose(expose);
</script>
