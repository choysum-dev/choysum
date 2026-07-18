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
    :action-ids="{ create: countryActions.create, delete: countryActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Code" :label="_t('Code')" />
    <OVarCharField :store="store" prop="PhonePrefix" :label="_t('Dialing Code')" />
    <OManyToOneField :store="store" prop="DefaultCurrencyId" :label="_t('Default Currency')"
      ><OVarCharField :store="store" prop="DefaultCurrencyId.Name" :label="_t('Currency')"
    /></OManyToOneField>
    <OBooleanField :store="store" prop="ZipRequired" :label="_t('ZIP Required')" />
    <OBooleanField :store="store" prop="StateRequired" :label="_t('State/Province Required')" />
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Country from '@/base/service/models/country';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'CountryListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/CountryListView' });
const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/CountryListView' });
const props = defineProps<{ store: WebModelStore<Country> }>();
const countryActions = defineModelActions('base.Country', { entityTitle: _tRef('Country') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<Country>) {
  router.push(`/base/countries/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<Country>();
defineExpose(expose);
</script>
