<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: bankActions.create, edit: bankActions.edit, copy: bankActions.copy, delete: bankActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <el-card shadow="never" class="bfv-card">
      <template #header
        ><div class="bfv-card__header"><span>{{ _t('Bank Information') }}</span></div></template
      >
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Name" :rules="requiredRules" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="Code" /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OVarCharField :store="store" prop="BIC" /></el-col>
      </el-row>
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField :store="store" prop="CountryId" :search-view="CountryListView" :search-view-title="_t('Select Country')"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"
          ><OManyToOneField
            :store="store"
            prop="AddressId"
            :search-view="AddressListView"
            :search-view-title="_t('Select Address')"
            @value-click="onAddressValueClick"
        /></el-col>
        <el-col :xs="24" :sm="12" :md="8"><OBooleanField :store="store" prop="IsActive" /></el-col>
      </el-row>
    </el-card>
  </OFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Bank from '@/base/service/models/bank';
import type Address from '@/base/service/models/address';
import { ElCard, ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import CountryListView from './CountryListView.vue';
import AddressListView from './AddressListView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'BankFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/BankFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store: WebModelStore<Bank>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const bankActions = defineModelActions('base.Bank', { entityTitle: _lt('Bank') });
const { hasAction } = usePermission();
const { store, recordId, viewMode, showHeader, createAction } = props;
const router = useRouter();

function onAddressValueClick(payload: ManyToOneValueClickPayload<Address>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'AddressDetail', params: { id } });
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
