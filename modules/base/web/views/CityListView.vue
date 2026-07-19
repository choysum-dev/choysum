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
    :action-ids="{ create: cityActions.create, delete: cityActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Code" :label="_t('Code')" />
    <OManyToOneField :store="store" prop="CountryId" :label="_t('Country')"><OVarCharField :store="store" prop="CountryId.Name" :label="_t('Country')" /></OManyToOneField>
    <OManyToOneField :store="store" prop="StateId" :label="_t('State/Province')"
      ><OVarCharField :store="store" prop="StateId.Name" :label="_t('State/Province')"
    /></OManyToOneField>
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type City from '@/base/service/models/city';
import { useRouter } from 'vue-router';
import OListView from '@/web/web/components/view/OListView.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'CityListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/CityListView' });
const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/CityListView' });
const props = defineProps<{ store: WebModelStore<City> }>();
const cityActions = defineModelActions('base.City', { entityTitle: _tRef('City') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<City>) {
  router.push(`/base/cities/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<City>();
defineExpose(expose);
</script>
