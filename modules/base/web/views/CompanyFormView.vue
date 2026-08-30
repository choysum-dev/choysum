<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{ create: companyActions.create, edit: companyActions.edit, copy: companyActions.copy, delete: companyActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header>
        <div class="bfv-card__header"><span>{{ _t('Basic Information') }}</span></div>
      </template>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Name" :rules="requiredRules" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OVarCharField :store="store" prop="Code" :rules="requiredRules" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OSelectionField
            :store="store"
            prop="Timezone"
            :rules="requiredRules"
            :placeholder="_t('Select a time zone')"
            :select-props="{ filterable: true, allowCreate: false }"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneField
            :store="store"
            prop="CurrencyId"
            :rules="requiredRules"
            :search-view="CurrencyListView"
            :search-view-title="_t('Select Currency')"
            @value-click="onCurrencyValueClick"
          />
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <OManyToOneField
            :store="store"
            prop="ParentId"
            :search-view="CompanyListView"
            :search-view-title="_t('Select Parent Company')"
            @value-click="onParentCompanyValueClick"
          />
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="CreatedAt" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="6">
          <ODateTimeField :store="store" prop="UpdatedAt" />
        </el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Company from '@/base/service/models/company';
import type Currency from '@/base/service/models/currency';
import { ElCard, ElRow, ElCol } from 'element-plus';

import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CompanyListView from './CompanyListView.vue';
import CurrencyListView from './CurrencyListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'CompanyFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CompanyFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Company>;
    recordId?: string;
    initialValues?: Partial<Company>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  }
);

const companyActions = defineModelActions('base.Company', { entityTitle: _lt('Company') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'CompanyFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const { initialValues } = props;
const router = useRouter();

function onParentCompanyValueClick(payload: ManyToOneValueClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}

function onCurrencyValueClick(payload: ManyToOneValueClickPayload<Currency>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CurrencyDetail', params: { id } });
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
