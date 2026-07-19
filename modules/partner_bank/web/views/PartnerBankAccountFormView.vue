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
    v-slot="{ viewMode: formViewMode }"
    v-on="$attrs"
  >
    <div class="pbafv-grid">
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OManyToOneRefField
            :store="store"
            prop="BankId"
            :label="_t('Bank')"
            :searchView="BankListView"
            :search-view-title="_t('Select Bank')"
            @value-click="onBankValueClick"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="AccountName" :label="_t('Account Name')" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="AccountNo" :label="_t('Account Number')" :visible="formViewMode !== 'display'" />
          <OVarCharField :store="store" prop="AccountNoMasked" :label="_t('Account Number')" :visible="formViewMode === 'display'" :readonly="true" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OSelectionField :store="store" prop="AccountType" :label="_t('Account Type')" :selection="accountTypeOptions" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="IBAN" :label="_t('IBAN')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="RoutingCode" :label="_t('Routing Code')" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="AllowInbound" :label="_t('Allow Inbound')" widget="switch" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="AllowOutbound" :label="_t('Allow Outbound')" widget="switch" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsDefaultInbound" :label="_t('Default Inbound')" widget="switch" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsDefaultOutbound" :label="_t('Default Outbound')" widget="switch" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
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
import type Bank from '@/base/service/models/bank';
import { ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import BankListView from '@/base/web/views/BankListView.vue';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'PartnerBankAccountFormView', inheritAttrs: true });
const { _t } = createTranslate('partner_bank', { scope: 'web/views/PartnerBankAccountFormView' });

/**
 * Props consumed by the partner bank account form view.
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
 * Account type options supported by partner bank accounts.
 */
const accountTypeOptions = ['checking', 'savings', 'corporate', 'other'];

/**
 * Opens the selected bank record from the bank account form.
 */
function onBankValueClick(payload: ManyToOneRefValueClickPayload<Bank>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'BankDetail', params: { id } });
}
</script>

<style scoped>
.pbafv-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
