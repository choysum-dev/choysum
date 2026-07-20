<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: addressActions.create, edit: addressActions.edit, copy: addressActions.copy, delete: addressActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('Address Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Label" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Zip" /></el-col>
        <el-col :xs="24" :sm="12" :md="8">
          <OManyToOneField
            :store="store"
            prop="CountryId"
            :search-view="CountryListView"
            :search-view-title="_t('Select Country')"
            @value-click="onCountryValueClick"
          />
        </el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12"><OVarCharField :store="store" prop="Street1" /></el-col>
        <el-col :xs="24" :sm="12"><OVarCharField :store="store" prop="Street2" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="StateId"
            :search-view="StateListView"
            :search-view-title="_t('Select State/Province')"
            @value-click="onStateValueClick"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="CityId"
            :search-view="CityListView"
            :search-view-title="_t('Select City')"
            @value-click="onCityValueClick"
        /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Address from '@/base/service/models/address';
import type Country from '@/base/service/models/country';
import type State from '@/base/service/models/state';
import type City from '@/base/service/models/city';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CountryListView from './CountryListView.vue';
import StateListView from './StateListView.vue';
import CityListView from './CityListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'AddressFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/AddressFormView' });

const props = withDefaults(
  defineProps<{
    store: WebModelStore<Address>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  { showHeader: true, createAction: undefined }
);

const addressActions = defineModelActions('base.Address', { entityTitle: _lt('Address') });
const { hasAction } = usePermission();
const { store, recordId, viewMode, showHeader, createAction } = props;
const router = useRouter();

function onCountryValueClick(payload: ManyToOneValueClickPayload<Country>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CountryDetail', params: { id } });
}

function onStateValueClick(payload: ManyToOneValueClickPayload<State>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'StateDetail', params: { id } });
}

function onCityValueClick(payload: ManyToOneValueClickPayload<City>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CityDetail', params: { id } });
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
