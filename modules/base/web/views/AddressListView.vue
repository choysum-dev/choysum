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
    :action-ids="{ create: addressActions.create, delete: addressActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Label" :label="_t('Label')" />
    <OVarCharField :store="store" prop="Street1" :label="_t('Address Line 1')" />
    <OVarCharField :store="store" prop="Zip" :label="_t('ZIP')" />
    <OManyToOneField :store="store" prop="CountryId" :label="_t('Country')"><OVarCharField :store="store" prop="CountryId.Name" :label="_t('Country')" /></OManyToOneField>
    <OManyToOneField :store="store" prop="StateId" :label="_t('State/Province')"
      ><OVarCharField :store="store" prop="StateId.Name" :label="_t('State/Province')"
    /></OManyToOneField>
    <OManyToOneField :store="store" prop="CityId" :label="_t('City')"><OVarCharField :store="store" prop="CityId.Name" :label="_t('City')" /></OManyToOneField>
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Address from '@/base/service/models/address';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'AddressListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/AddressListView' });
const props = defineProps<{ store: WebModelStore<Address> }>();
const addressActions = defineModelActions('base.Address', { entityTitle: 'Address' });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<Address>) {
  router.push(`/base/addresses/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<Address>();
defineExpose(expose);
</script>
