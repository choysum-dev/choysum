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
    <div class="pcifv-grid">
      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="IdentifierType" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="Value" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyManyToOneField
            :store="store"
            prop="CountryId"
            :searchView="CountryListView"
            :search-view-title="_t('Select Country')"
            @value-click="onCountryValueClick"
          />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyVarcharField :store="store" prop="IssuedBy" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="ValidFrom" mode="datetime" />
        </ChoyCol>
        <ChoyCol :span="6">
          <ChoyDatetimeField :store="store" prop="ValidTo" mode="datetime" />
        </ChoyCol>
      </ChoyGrid>

      <ChoyGrid :cols="12">
        <ChoyCol :span="6">
          <ChoyBooleanField :store="store" prop="IsPrimary" widget="switch" />
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
import type Country from '@/base/service/models/country';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CountryListView from '@/base/web/views/CountryListView.vue';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyCol,
  ChoyDatetimeField,
  ChoyFormView,
  ChoyGrid,
  ChoyManyToOneField,
  ChoyVarcharField,
  type ChoyViewMode as ViewMode,
} from '@/web';

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
function onCountryValueClick(payload: ManyToOneValueClickPayload<Country>) {
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
