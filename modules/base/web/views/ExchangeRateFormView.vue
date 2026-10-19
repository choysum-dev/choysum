<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: exchangeRateActions.create, edit: exchangeRateActions.edit, copy: exchangeRateActions.copy, delete: exchangeRateActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Exchange Rate Information')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="CurrencyId"
            :search-view="CurrencyListView"
            :search-view-title="_t('Select Currency')"
            @value-click="onCurrencyValueClick"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="CompanyId"
            :search-view="CompanyListView"
            :search-view-title="_t('Select Company')"
            @value-click="onCompanyValueClick"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyDateField :store="store" prop="Date" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyDecimalField :store="store" prop="Rate" :rules="requiredRules"
        /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type ExchangeRate from '@/base/service/models/exchange_rate';
import type Currency from '@/base/service/models/currency';
import type Company from '@/base/service/models/company';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CurrencyListView from './CurrencyListView.vue';
import CompanyListView from './CompanyListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyCard, ChoyCol, ChoyDateField, ChoyDecimalField, ChoyFormView, ChoyGrid, ChoyManyToOneField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'ExchangeRateFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/ExchangeRateFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{
    store?: WebModelStore<ExchangeRate>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  { showHeader: true, createAction: undefined }
);
const exchangeRateActions = defineModelActions('base.ExchangeRate', { entityTitle: _lt('Exchange Rate') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'ExchangeRateFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const router = useRouter();

function onCurrencyValueClick(payload: ManyToOneValueClickPayload<Currency>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CurrencyDetail', params: { id } });
}

function onCompanyValueClick(payload: ManyToOneValueClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}
</script>

