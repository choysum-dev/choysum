<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    :store="store"
    :record-id="recordId"
    :initial-values="initialValues"
    :view-mode="viewMode"
    :show-header="showHeader"
    :show-actions="showActions"
    :show-messages="showMessages"
    v-on="$attrs"
  >
    <div class="pcifv-grid">
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="IdentifierType" :label="_t('Type')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="Value" :label="_t('Value')" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OManyToOneRefField
            :store="store"
            prop="CountryId"
            :label="_t('Country')"
            :searchView="CountryListView"
            :search-view-title="_t('Select Country')"
            @value-click="onCountryValueClick"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="IssuedBy" :label="_t('Issued By')" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <ODateTimeField :store="store" prop="ValidFrom" :label="_t('Valid From')" mode="datetime" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <ODateTimeField :store="store" prop="ValidTo" :label="_t('Valid To')" mode="datetime" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsPrimary" :label="_t('Primary')" widget="switch" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" widget="switch" />
        </el-col>
      </el-row>
    </div>
  </OFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import type Country from '@/base/service/models/country';
import { ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import CountryListView from '@/base/web/views/CountryListView.vue';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'PartnerIdentifierFormView', inheritAttrs: true });
const { _t } = createTranslate('partner_commercial', { scope: 'web/views/PartnerIdentifierFormView' });

/**
 * Props consumed by the partner identifier form view.
 */
const props = withDefaults(
  defineProps<{
    store: WebModelStore<any>;
    recordId?: string;
    initialValues?: Record<string, any>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    showActions?: boolean;
    showMessages?: boolean;
  }>(),
  {
    viewMode: 'create',
  }
);

const { store, recordId, initialValues, viewMode, showHeader, showActions, showMessages } = props;
const router = useRouter();

/**
 * Opens the selected country record from the identifier form.
 */
function onCountryValueClick(payload: ManyToOneRefValueClickPayload<Country>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CountryDetail', params: { id } });
}
</script>

<style scoped>
.pcifv-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
