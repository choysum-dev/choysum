<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :action-ids="{ create: exchangeRateActions.create, delete: exchangeRateActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyManyToOneField :store="store" prop="CurrencyId"
      ><ChoyVarcharField :store="store" prop="CurrencyId.Name"
    /></ChoyManyToOneField>
    <ChoyManyToOneField :store="store" prop="CompanyId"><ChoyVarcharField :store="store" prop="CompanyId.Name" /></ChoyManyToOneField>
    <ChoyDateField :store="store" prop="Date" />
    <ChoyNumberField :store="store" prop="Rate" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type ExchangeRate from '@/base/service/models/exchange_rate';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';
import { ChoyDateField, ChoyListView, ChoyManyToOneField, ChoyNumberField, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';

defineOptions({ name: 'ExchangeRateListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/ExchangeRateListView' });
const props = defineProps<{ store?: WebModelStore<ExchangeRate> }>();
const store = resolvePageStore(props.store, 'ExchangeRateListView');
const exchangeRateActions = defineModelActions('base.ExchangeRate', { entityTitle: _lt('Exchange Rate') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/base/exchange-rates/${id}`);
}
const { listRef, expose } = useListViewExpose<ExchangeRate>();
defineExpose(expose);
</script>
