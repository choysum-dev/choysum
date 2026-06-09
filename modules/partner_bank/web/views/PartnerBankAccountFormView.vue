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
            label="银行"
            :searchView="BankListView"
            search-view-title="选择银行"
            @value-click="onBankValueClick"
          />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="AccountName" label="账户名称" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="AccountNo" label="账号" :visible="formViewMode !== 'display'" />
          <OVarCharField :store="store" prop="AccountNoMasked" label="账号" :visible="formViewMode === 'display'" :readonly="true" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OSelectionField :store="store" prop="AccountType" label="账户类型" :selection="accountTypeOptions" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="IBAN" label="IBAN" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="RoutingCode" label="路由码" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="AllowInbound" label="允许入账" widget="switch" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="AllowOutbound" label="允许出账" widget="switch" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsDefaultInbound" label="默认入账" widget="switch" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsDefaultOutbound" label="默认出账" widget="switch" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsActive" label="启用" widget="switch" />
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
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/OManyToOneRefField.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import BankListView from '@/base/web/views/BankListView.vue';

defineOptions({ name: 'PartnerBankAccountFormView', inheritAttrs: true });

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
