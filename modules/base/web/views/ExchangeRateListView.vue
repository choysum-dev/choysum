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
    :action-ids="{ create: exchangeRateActions.create, delete: exchangeRateActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OManyToOneField :store="store" prop="CurrencyId"
      ><OVarCharField :store="store" prop="CurrencyId.Name"
    /></OManyToOneField>
    <OManyToOneField :store="store" prop="CompanyId"><OVarCharField :store="store" prop="CompanyId.Name" /></OManyToOneField>
    <ODateField :store="store" prop="Date" />
    <ODecimalField :store="store" prop="Rate" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type ExchangeRate from '@/base/service/models/exchange_rate';
import { useRouter } from 'vue-router';
import OListView from '@/web/web/components/view/OListView.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateField from '@/web/web/components/field/ODateField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'ExchangeRateListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/ExchangeRateListView' });
const props = defineProps<{ store?: WebModelStore<ExchangeRate> }>();
const store = resolvePageStore(props.store, 'ExchangeRateListView');
const exchangeRateActions = defineModelActions('base.ExchangeRate', { entityTitle: _lt('Exchange Rate') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<ExchangeRate>) {
  router.push(`/base/exchange-rates/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<ExchangeRate>();
defineExpose(expose);
</script>
