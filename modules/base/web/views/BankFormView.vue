<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: bankActions.create, edit: bankActions.edit, copy: bankActions.copy, delete: bankActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Bank Information')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Code" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="BIC" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyManyToOneField :store="store" prop="CountryId" :search-view="CountryListView" :search-view-title="_t('Select Country')"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="AddressId"
            :search-view="AddressListView"
            :search-view-title="_t('Select Address')"
            @value-click="onAddressValueClick"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Bank from '@/base/service/models/bank';
import type Address from '@/base/service/models/address';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CountryListView from './CountryListView.vue';
import AddressListView from './AddressListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyVarcharField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'BankFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/BankFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: WebModelStore<Bank>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const bankActions = defineModelActions('base.Bank', { entityTitle: _lt('Bank') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'BankFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const router = useRouter();

function onAddressValueClick(payload: ManyToOneValueClickPayload<Address>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'AddressDetail', params: { id } });
}
</script>

