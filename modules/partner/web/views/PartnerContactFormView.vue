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
    <div class="pcfv-grid">
      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OSelectionField :store="store" prop="ContactRole" :label="_t('Role')" :selection="contactRoleOptions" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OSelectionField :store="store" prop="AddressType" :label="_t('Address Type')" :selection="addressTypeOptions" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OManyToOneRefField
            :store="store"
            prop="AddressId"
            :label="_t('Address')"
            :searchView="AddressListView"
            :search-view-title="_t('Select Address')"
            @value-click="onAddressValueClick"
          />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="Email" :label="_t('Email')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="Phone" :label="_t('Phone')" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OVarCharField :store="store" prop="Mobile" :label="_t('Mobile')" />
        </el-col>
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OIntField :store="store" prop="Sequence" :label="_t('Sequence')" />
        </el-col>
      </el-row>

      <el-row :gutter="12">
        <el-col :xs="24" :sm="12" :md="12" :lg="12">
          <OBooleanField :store="store" prop="IsDefault" :label="_t('Default')" widget="switch" />
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
import type Address from '@/base/service/models/address';
import { ElRow, ElCol } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import AddressListView from '@/base/web/views/AddressListView.vue';
import { createTranslate } from '@/web/web/i18n';

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
function onAddressValueClick(payload: ManyToOneRefValueClickPayload<Address>) {
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
