<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{ create: partnerActions.create, edit: partnerActions.edit, copy: partnerActions.copy, delete: partnerActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <div data-region="partner-detail-root">
      <div data-region="partner-primary-form">
        <el-card shadow="never" class="pfv-card" data-region="partner-section-basic">
          <template #header>
            <div class="pfv-card__header"><span>{{ _t('Basic Information') }}</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Name" :label="_t('Name')" :rules="requiredRules" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Code" :label="_t('Code')" :rules="requiredRules" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField
                :store="store"
                prop="CompanyId"
                :label="_t('Company')"
                :searchView="CompanyListView"
                :search-view-title="_t('Select Company')"
                :rules="requiredRules"
                @value-click="onCompanyValueClick"
              />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OBooleanField :store="store" prop="IsCompany" :label="_t('Organization')" widget="switch" />
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" widget="switch" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <ODateTimeField :store="store" prop="CreatedAt" :label="_t('Created At')" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <ODateTimeField :store="store" prop="UpdatedAt" :label="_t('Updated At')" />
            </el-col>
          </el-row>
        </el-card>

        <el-card shadow="never" class="pfv-card" data-region="partner-section-commercial-basics">
          <template #header>
            <div class="pfv-card__header"><span>{{ _t('Commercial Basics') }}</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OIntField :store="store" prop="CustomerRank" :label="_t('Customer Rank')" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OIntField :store="store" prop="SupplierRank" :label="_t('Supplier Rank')" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField :store="store" prop="LanguageId" :label="_t('Default Language')" :searchView="LanguageListView" :search-view-title="_t('Select Language')" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField :store="store" prop="CurrencyId" :label="_t('Default Currency')" :searchView="CurrencyListView" :search-view-title="_t('Select Currency')" />
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OManyToOneRefField :store="store" prop="CountryId" :label="_t('Country')" :searchView="CountryListView" :search-view-title="_t('Select Country')" />
            </el-col>
          </el-row>
        </el-card>

        <el-card shadow="never" class="pfv-card" data-region="partner-section-contact-channel">
          <template #header>
            <div class="pfv-card__header"><span>{{ _t('Contact Details') }}</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Email" :label="_t('Email')" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Phone" :label="_t('Phone')" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Mobile" :label="_t('Mobile')" />
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="6">
              <OVarCharField :store="store" prop="Reference" :label="_t('External Reference')" />
            </el-col>
          </el-row>
        </el-card>

        <el-card shadow="never" class="pfv-card" data-region="partner-section-default-entry">
          <template #header>
            <div class="pfv-card__header"><span>{{ _t('Default Entries') }}</span></div>
          </template>
          <el-row :gutter="12">
            <el-col :xs="24" :sm="12" :md="8" :lg="8">
              <OManyToOneField :store="store" prop="DefaultContactId" :label="_t('Default Contact')" :readonly="true" @value-click="onDefaultContactValueClick">
                <OVarCharField :store="store" prop="DefaultContactId.Name" :label="_t('Name')" />
              </OManyToOneField>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="8">
              <OManyToOneField
                :store="store"
                prop="DefaultBillingAddressId"
                :label="_t('Default Billing Address')"
                :readonly="true"
                @value-click="onDefaultBillingAddressValueClick"
              >
                <OVarCharField :store="store" prop="DefaultBillingAddressId.Name" :label="_t('Name')" />
              </OManyToOneField>
            </el-col>
            <el-col :xs="24" :sm="12" :md="8" :lg="8">
              <OManyToOneField
                :store="store"
                prop="DefaultShippingAddressId"
                :label="_t('Default Shipping Address')"
                :readonly="true"
                @value-click="onDefaultShippingAddressValueClick"
              >
                <OVarCharField :store="store" prop="DefaultShippingAddressId.Name" :label="_t('Name')" />
              </OManyToOneField>
            </el-col>
          </el-row>
        </el-card>
      </div>

      <el-card shadow="never" class="pfv-card" data-region="partner-detail-tabs">
        <template #header>
          <div class="pfv-card__header">
            <span>{{ _t('Related Data') }}</span>
            <div data-slot="partner-detail-tab-list" style="display: none"></div>
          </div>
        </template>
        <el-tabs v-model="activeTab" type="card" class="pfv-tabs" data-slot="partner-detail-tab-panels">
          <el-tab-pane :label="_t('Contacts and Addresses')" name="contacts" data-region="partner-tab-contacts">
            <div data-region="partner-panel-contacts">
              <OOneToManyKanbanField
                :store="store"
                prop="Contacts"
                label=""
                :default-record="defaultContactRecord"
                :editable="canEditContacts()"
                :removable="canEditContacts()"
                :add-button-text="_t('Add Contact')"
                :form-view="PartnerContactFormView"
                :create-dialog-title="_t('New Contact')"
                :edit-dialog-title="_t('Edit Contact')"
                :display-dialog-title="_t('View Contact')"
              >
                <template #card="{ item, editable, removable, edit, remove }">
                  <div class="pfv-contact-card">
                    <div class="pfv-contact-card__title-row">
                      <div class="pfv-contact-card__title">{{ item?.Name || _t('Unnamed Contact') }}</div>
                      <div class="pfv-contact-card__flags">
                        <el-tag v-if="item?.IsDefault" size="small" type="success">{{ _t('Default') }}</el-tag>
                        <el-tag v-if="item?.IsActive === false" size="small" type="info">{{ _t('Inactive') }}</el-tag>
                      </div>
                    </div>
                    <div class="pfv-contact-card__meta">{{ _t('Role') }}: {{ getContactRoleLabel(item?.ContactRole) }}</div>
                    <div class="pfv-contact-card__meta">{{ _t('Address Type') }}: {{ getAddressTypeLabel(item?.AddressType) }}</div>
                    <div v-if="item?.Email" class="pfv-contact-card__line">{{ _t('Email') }}: {{ item.Email }}</div>
                    <div v-if="item?.Phone" class="pfv-contact-card__line">{{ _t('Phone') }}: {{ item.Phone }}</div>
                    <div v-if="editable || removable" class="pfv-contact-card__actions">
                      <el-button v-if="editable" type="primary" text size="small" @click.stop="edit">{{ _t('Edit') }}</el-button>
                      <el-button v-if="removable" type="danger" text size="small" @click.stop="remove">{{ _t('Delete') }}</el-button>
                    </div>
                  </div>
                </template>
              </OOneToManyKanbanField>
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-card>
    </div>
  </OFormView>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Partner from '@/partner/service/models/partner';
import type Company from '@/base/service/models/company';
import type Address from '@/base/service/models/address';
import { ElCard, ElRow, ElCol, ElTabs, ElTabPane } from 'element-plus';
import OFormView from '@/web/web/components/view/OFormView.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OIntField from '@/web/web/components/field/OIntField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OManyToOneRefField from '@/web/web/components/field/OManyToOneRefField.vue';
import type { ValueClickPayload as ManyToOneRefValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import OOneToManyKanbanField from '@/web/web/components/field/OOneToManyKanbanField.vue';
import CompanyListView from '@/base/web/views/CompanyListView.vue';
import LanguageListView from '@/base/web/views/LanguageListView.vue';
import CurrencyListView from '@/base/web/views/CurrencyListView.vue';
import CountryListView from '@/base/web/views/CountryListView.vue';
import PartnerContactFormView from '@/partner/web/views/PartnerContactFormView.vue';
import type { ViewMode } from '@/web/web/components/view/OViewScope.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ElButton, ElTag } from 'element-plus';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'PartnerFormView', inheritAttrs: true });
const { _t } = createTranslate('partner', { scope: 'web/views/PartnerFormView' });
const { _t: _tRef } = createTranslate('partner', { output: 'reference', scope: 'web/views/PartnerFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

/**
 * Props consumed by the partner form view.
 */
const props = withDefaults(
  defineProps<{
    store: WebModelStore<Partner>;
    recordId?: string;
    initialValues?: Partial<Partner>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  }
);

const { store, recordId, initialValues, viewMode, showHeader, createAction } = props;
const partnerActions = defineModelActions('partner.Partner', { entityTitle: _tRef('Partner') });
const { hasAction } = usePermission();
const router = useRouter();
const activeTab = ref('contacts');

/**
 * Builds the default contact row used when a new related contact is added.
 */
function defaultContactRecord() {
  return {
    IsActive: true,
    IsDefault: false,
    Sequence: 10,
  };
}

/**
 * Reports whether the current actor can edit related contact rows.
 */
function canEditContacts() {
  return hasAction(partnerActions.edit);
}

/**
 * Maps a contact address type to its display label.
 */
function getAddressTypeLabel(value?: string) {
  switch (value) {
    case 'billing':
      return _t('Billing Address');
    case 'shipping':
      return _t('Shipping Address');
    case 'office':
      return _t('Office Address');
    case 'registered':
      return _t('Registered Address');
    case 'other':
      return _t('Other');
    default:
      return value || '-';
  }
}

/**
 * Maps a contact role to its display label.
 */
function getContactRoleLabel(value?: string) {
  switch (value) {
    case 'general':
      return _t('General Contact');
    case 'billing':
      return _t('Billing Contact');
    case 'shipping':
      return _t('Shipping Contact');
    case 'procurement':
      return _t('Procurement Contact');
    case 'sales':
      return _t('Sales Contact');
    case 'finance':
      return _t('Finance Contact');
    case 'legal':
      return _t('Legal Contact');
    default:
      return value || '-';
  }
}

/**
 * Opens the selected company record from the partner form.
 */
function onCompanyValueClick(payload: ManyToOneRefValueClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}

/**
 * Opens the derived default contact record.
 */
function onDefaultContactValueClick(payload: ManyToOneValueClickPayload<Partner>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'PartnerDetail', params: { id } });
}

/**
 * Opens the derived default billing address record.
 */
function onDefaultBillingAddressValueClick(payload: ManyToOneValueClickPayload<Address>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'AddressDetail', params: { id } });
}

/**
 * Opens the derived default shipping address record.
 */
function onDefaultShippingAddressValueClick(payload: ManyToOneValueClickPayload<Address>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'AddressDetail', params: { id } });
}
</script>

<style scoped>
.pfv-card {
  margin-bottom: 14px;
}

.pfv-card__header {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.pfv-tabs {
  --el-tabs-header-height: 42px;
}

.pfv-contact-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  height: 100%;
  min-height: 0;
}

.pfv-contact-card__title-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.pfv-contact-card__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.pfv-contact-card__flags {
  display: inline-flex;
  gap: 6px;
}

.pfv-contact-card__meta,
.pfv-contact-card__line {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.pfv-contact-card__actions {
  margin-top: auto;
  padding-top: 4px;
  display: inline-flex;
  gap: 4px;
}
</style>
