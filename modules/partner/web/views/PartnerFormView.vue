<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{ create: partnerActions.create, edit: partnerActions.edit, copy: partnerActions.copy, delete: partnerActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <div data-region="partner-detail-root">
      <div data-region="partner-primary-form">
        <ChoyCard :title="_t('Basic Information')" class="mb-3.5" data-region="partner-section-basic">
          <ChoyGrid :cols="12">
            <ChoyCol :span="3">
              <ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyManyToOneRefField
                :store="store"
                prop="CompanyId"
                :searchView="CompanyListView"
                :search-view-title="_t('Select Company')"
                :rules="requiredRules"
                @value-click="onCompanyValueClick"
              />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyBooleanField :store="store" prop="IsCompany" widget="switch" />
            </ChoyCol>
          </ChoyGrid>
          <ChoyGrid :cols="12">
            <ChoyCol :span="3">
              <ChoyBooleanField :store="store" prop="IsActive" widget="switch" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyDatetimeField :store="store" prop="CreatedAt" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyDatetimeField :store="store" prop="UpdatedAt" />
            </ChoyCol>
          </ChoyGrid>
        </ChoyCard>

        <ChoyCard :title="_t('Commercial Basics')" class="mb-3.5" data-region="partner-section-commercial-basics">
          <ChoyGrid :cols="12">
            <ChoyCol :span="3">
              <ChoyIntField :store="store" prop="CustomerRank" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyIntField :store="store" prop="SupplierRank" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyManyToOneRefField
                :store="store"
                prop="LanguageId"
                :searchView="LanguageListView"
                :search-view-title="_t('Select Language')"
              />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyManyToOneRefField
                :store="store"
                prop="CurrencyId"
                :searchView="CurrencyListView"
                :search-view-title="_t('Select Currency')"
              />
            </ChoyCol>
          </ChoyGrid>
          <ChoyGrid :cols="12">
            <ChoyCol :span="3">
              <ChoyManyToOneRefField
                :store="store"
                prop="CountryId"
                :searchView="CountryListView"
                :search-view-title="_t('Select Country')"
              />
            </ChoyCol>
          </ChoyGrid>
        </ChoyCard>

        <ChoyCard :title="_t('Contact Details')" class="mb-3.5" data-region="partner-section-contact-channel">
          <ChoyGrid :cols="12">
            <ChoyCol :span="3">
              <ChoyVarcharField :store="store" prop="Email" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyVarcharField :store="store" prop="Phone" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyVarcharField :store="store" prop="Mobile" />
            </ChoyCol>
            <ChoyCol :span="3">
              <ChoyVarcharField :store="store" prop="Reference" />
            </ChoyCol>
          </ChoyGrid>
        </ChoyCard>

        <ChoyCard :title="_t('Default Entries')" class="mb-3.5" data-region="partner-section-default-entry">
          <ChoyGrid :cols="12">
            <ChoyCol :span="4">
              <ChoyManyToOneField
                :store="store"
                prop="DefaultContactId"
                :readonly="true"
                @value-click="onDefaultContactValueClick"
              >
                <ChoyVarcharField :store="store" prop="DefaultContactId.Name" />
              </ChoyManyToOneField>
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyManyToOneField
                :store="store"
                prop="DefaultBillingAddressId"
                :readonly="true"
                @value-click="onDefaultBillingAddressValueClick"
              >
                <ChoyVarcharField :store="store" prop="DefaultBillingAddressId.Name" />
              </ChoyManyToOneField>
            </ChoyCol>
            <ChoyCol :span="4">
              <ChoyManyToOneField
                :store="store"
                prop="DefaultShippingAddressId"
                :readonly="true"
                @value-click="onDefaultShippingAddressValueClick"
              >
                <ChoyVarcharField :store="store" prop="DefaultShippingAddressId.Name" />
              </ChoyManyToOneField>
            </ChoyCol>
          </ChoyGrid>
        </ChoyCard>
      </div>

      <ChoyCard :title="_t('Related Data')" class="mb-3.5" data-region="partner-detail-tabs">
        <ChoyTabs
          v-model="activeTab"
          class="mt-0"
          :data-anchor="PARTNER_DETAIL_TAB_PANELS_ANCHOR"
          default-value="contacts"
        >
          <ChoyTab value="contacts" :label="_t('Contacts and Addresses')" data-region="partner-tab-contacts">
            <div data-region="partner-panel-contacts">
              <ChoyOneToManyKanbanField
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
                  <div class="flex h-full min-h-0 flex-col gap-1.5">
                    <div class="flex items-center justify-between gap-2">
                      <div class="text-sm font-semibold text-foreground">{{ item?.Name || _t('Unnamed Contact') }}</div>
                      <div class="inline-flex gap-1.5">
                        <span v-if="item?.IsDefault" class="inline-flex items-center rounded-md border border-transparent px-2 py-0.5 text-xs font-semibold bg-success/20 text-success">{{ _t('Default') }}</span>
                        <span v-if="item?.IsActive === false" class="inline-flex items-center rounded-md border border-transparent px-2 py-0.5 text-xs font-semibold bg-muted text-muted-foreground">{{ _t('Inactive') }}</span>
                      </div>
                    </div>
                    <div class="text-xs text-muted-foreground">{{ _t('Role') }}: {{ getContactRoleLabel(item?.ContactRole) }}</div>
                    <div class="text-xs text-muted-foreground">{{ _t('Address Type') }}: {{ getAddressTypeLabel(item?.AddressType) }}</div>
                    <div v-if="item?.Email" class="text-xs text-muted-foreground">{{ _t('Email') }}: {{ item.Email }}</div>
                    <div v-if="item?.Phone" class="text-xs text-muted-foreground">{{ _t('Phone') }}: {{ item.Phone }}</div>
                    <div v-if="editable || removable" class="mt-auto inline-flex gap-1 pt-1">
                      <ChoyButton v-if="editable" variant="ghost" size="sm" @click.stop="edit">{{ _t('Edit') }}</ChoyButton>
                      <ChoyButton v-if="removable" variant="ghost" size="sm" @click.stop="remove">{{ _t('Delete') }}</ChoyButton>
                    </div>
                  </div>
                </template>
              </ChoyOneToManyKanbanField>
            </div>
          </ChoyTab>
        </ChoyTabs>
      </ChoyCard>

      <ChoyChatter v-if="props.recordId" :key="props.recordId" bind-store model="partner.Partner" :res-id="props.recordId" />
    </div>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Partner from '@/partner/service/models/partner';
import type Company from '@/base/service/models/company';
import type Address from '@/base/service/models/address';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CompanyListView from '@/base/web/views/CompanyListView.vue';
import LanguageListView from '@/base/web/views/LanguageListView.vue';
import CurrencyListView from '@/base/web/views/CurrencyListView.vue';
import CountryListView from '@/base/web/views/CountryListView.vue';
import PartnerContactFormView from '@/partner/web/views/PartnerContactFormView.vue';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyButton,
  ChoyCard,
  ChoyChatter,
  ChoyCol,
  ChoyDatetimeField,
  ChoyFormView,
  ChoyGrid,
  ChoyIntField,
  ChoyManyToOneField,
  ChoyManyToOneRefField,
  ChoyOneToManyKanbanField,
  ChoyTab,
  ChoyTabs,
  ChoyVarcharField,
  PARTNER_DETAIL_TAB_PANELS_ANCHOR,
  type ChoyViewMode as ViewMode
} from '@/web';
import { partnerActions } from './partner_actions';

defineOptions({ name: 'PartnerFormView', inheritAttrs: true });
const { _t } = createTranslate('partner', { scope: 'web/views/PartnerFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

/**
 * Props consumed by the partner form view.
 */
const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Partner>;
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

const store = resolvePageStore(props.store, 'PartnerFormView');
const { recordId, initialValues, viewMode, showHeader, createAction } = props;
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
function onCompanyValueClick(payload: ManyToOneValueClickPayload<Company>) {
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

