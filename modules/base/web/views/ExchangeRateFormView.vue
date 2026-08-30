<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: exchangeRateActions.create, edit: exchangeRateActions.edit, copy: exchangeRateActions.copy, delete: exchangeRateActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('Exchange Rate Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="CurrencyId"
            :search-view="CurrencyListView"
            :search-view-title="_t('Select Currency')"
            @value-click="onCurrencyValueClick"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="CompanyId"
            :search-view="CompanyListView"
            :search-view-title="_t('Select Company')"
            @value-click="onCompanyValueClick"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><ODateField :store="store" prop="Date" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><ODecimalField :store="store" prop="Rate" :rules="requiredRules"
        /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type ExchangeRate from '@/base/service/models/exchange_rate';
import type Currency from '@/base/service/models/currency';
import type Company from '@/base/service/models/company';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import ODateField from '@/web/web/components/field/ODateField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import CurrencyListView from './CurrencyListView.vue';
import CompanyListView from './CompanyListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';

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

<style scoped>
.bfv-card {
  margin-bottom: 14px;
}
.bfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
</style>
