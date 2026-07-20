<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: cityActions.create, edit: cityActions.edit, copy: cityActions.copy, delete: cityActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('City Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" :rules="requiredRules" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Code" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="IsActive" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="CountryId"
            :search-view="CountryListView"
            :search-view-title="_t('Select Country')"
            @value-click="onCountryValueClick"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="StateId"
            :search-view="StateListView"
            :search-view-title="_t('Select State/Province')"
            @value-click="onStateValueClick"
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
import type City from '@/base/service/models/city';
import type Country from '@/base/service/models/country';
import type State from '@/base/service/models/state';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CountryListView from './CountryListView.vue';
import StateListView from './StateListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'CityFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CityFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store: WebModelStore<City>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const cityActions = defineModelActions('base.City', { entityTitle: _lt('City') });
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
