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
    :action-ids="{ create: currencyActions.create, delete: currencyActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Code" :label="_t('Code')" />
    <OVarCharField :store="store" prop="Symbol" :label="_t('Symbol')" />
    <OIntField :store="store" prop="DecimalDigits" :label="_t('Decimal Digits')" />
    <ODecimalField :store="store" prop="Rounding" :label="_t('Rounding Precision')" />
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Currency from '@/base/service/models/currency';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'CurrencyListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/CurrencyListView' });
const props = defineProps<{ store: WebModelStore<Currency> }>();
const currencyActions = defineModelActions('base.Currency', { entityTitle: 'Currency' });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<Currency>) {
  router.push(`/base/currencies/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<Currency>();
defineExpose(expose);
</script>
