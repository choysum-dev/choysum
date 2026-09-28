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
    v-on="$attrs"
  >
    <div class="pcfv-grid">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Name" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="ContactRole" :selection="contactRoleOptions" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoySelectionField :store="store" prop="AddressType" :selection="addressTypeOptions" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyManyToOneField
            :store="store"
            prop="AddressId"
            :searchView="AddressListView"
            :search-view-title="_t('Select Address')"
            @value-click="onAddressValueClick"
          />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Email" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Phone" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Mobile" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyNumberField :store="store" prop="Sequence" mode="integer" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsDefault" widget="switch" />
        </ChoyCol>
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
import type Address from '@/base/service/models/address';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import AddressListView from '@/base/web/views/AddressListView.vue';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyFormView,
  ChoyGrid,
  ChoyManyToOneField,
  ChoyNumberField,
  ChoySelectionField,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';

defineOptions({ name: 'PartnerContactFormView', inheritAttrs: true });
const { _t } = createTranslate('partner', { scope: 'web/views/PartnerContactFormView' });

/**
 * Props consumed by the partner contact form view.
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
 * Address type options supported by partner contacts.
 */
const addressTypeOptions = ['billing', 'shipping', 'office', 'registered', 'other'];

/**
 * Contact role options supported by partner contacts.
 */
const contactRoleOptions = ['general', 'billing', 'shipping', 'procurement', 'sales', 'finance', 'legal'];

/**
 * Opens the selected address record from the contact form.
 */
function onAddressValueClick(payload: ManyToOneValueClickPayload<Address>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'AddressDetail', params: { id } });
}
</script>

<style scoped>
.pcfv-grid {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
