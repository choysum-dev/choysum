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
    :action-ids="{ create: currencyActions.create, delete: currencyActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyVarcharField :store="store" prop="Symbol" />
    <ChoyNumberField :store="store" prop="DecimalDigits" />
    <ChoyNumberField :store="store" prop="Rounding" />
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Currency from '@/base/service/models/currency';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';
import { ChoyBooleanField, ChoyListView, ChoyNumberField, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';

defineOptions({ name: 'CurrencyListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CurrencyListView' });
const props = defineProps<{ store?: WebModelStore<Currency> }>();
const store = resolvePageStore(props.store, 'CurrencyListView');
const currencyActions = defineModelActions('base.Currency', { entityTitle: _lt('Currency') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/base/currencies/${id}`);
}
const { listRef, expose } = useListViewExpose<Currency>();
defineExpose(expose);
</script>
