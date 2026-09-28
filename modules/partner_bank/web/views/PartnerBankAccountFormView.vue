<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
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
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyManyToOneField
            :store="store"
            prop="BankId"
            :searchView="BankListView"
            :search-view-title="_t('Select Bank')"
            @value-click="onBankValueClick"
          />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="AccountName" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="AccountNo" :visible="formViewMode !== 'display'" />
          <ChoyVarcharField :store="store" prop="AccountNoMasked" :visible="formViewMode === 'display'" :readonly="true" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="AccountType" :selection="accountTypeOptions" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="IBAN" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="RoutingCode" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="AllowInbound" widget="switch" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="AllowOutbound" widget="switch" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsDefaultInbound" widget="switch" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsDefaultOutbound" widget="switch" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsActive" widget="switch" />
        </ChoyCol>
      </ChoyGrid>
    </div>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Bank from '@/base/service/models/bank';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import BankListView from '@/base/web/views/BankListView.vue';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoyManyToOneField,
  ChoySelectionField,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';

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
function onBankValueClick(payload: ManyToOneValueClickPayload<Bank>) {
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
