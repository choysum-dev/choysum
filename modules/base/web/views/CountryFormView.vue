<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: countryActions.create, edit: countryActions.edit, copy: countryActions.copy, delete: countryActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Country Information')" class="bfv-card"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="PhonePrefix" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="DefaultCurrencyId"
            :search-view="CurrencyListView"
            :search-view-title="_t('Select Currency')"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="ZipRequired" /></ChoyCol>
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="StateRequired" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
        <ChoyCol :span="12"><ChoyTextField :store="store" prop="AddressFormat" /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Country from '@/base/service/models/country';
import CurrencyListView from './CurrencyListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyTextField, ChoyVarcharField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'CountryFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CountryFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: WebModelStore<Country>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const countryActions = defineModelActions('base.Country', { entityTitle: _lt('Country') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'CountryFormView');
const { recordId, viewMode, showHeader, createAction } = props;
</script>

<style scoped>
.bfv-card {
  margin-bottom: 14px;
}
</style>
