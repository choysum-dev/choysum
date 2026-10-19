<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: addressActions.create, edit: addressActions.edit, copy: addressActions.copy, delete: addressActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Address Information')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Label" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Zip" /></ChoyCol>
        <ChoyCol :span="4">
          <ChoyManyToOneField
            :store="store"
            prop="CountryId"
            :search-view="CountryListView"
            :search-view-title="_t('Select Country')"
            @value-click="onCountryValueClick"
          />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="6"><ChoyVarcharField :store="store" prop="Street1" /></ChoyCol>
        <ChoyCol :span="6"><ChoyVarcharField :store="store" prop="Street2" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="StateId"
            :search-view="StateListView"
            :search-view-title="_t('Select State/Province')"
            @value-click="onStateValueClick"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="CityId"
            :search-view="CityListView"
            :search-view-title="_t('Select City')"
            @value-click="onCityValueClick"
        /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Address from '@/base/service/models/address';
import type Country from '@/base/service/models/country';
import type State from '@/base/service/models/state';
import type City from '@/base/service/models/city';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CountryListView from './CountryListView.vue';
import StateListView from './StateListView.vue';
import CityListView from './CityListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyVarcharField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'AddressFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/AddressFormView' });

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Address>;
    recordId?: string;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  { showHeader: true, createAction: undefined }
);

const addressActions = defineModelActions('base.Address', { entityTitle: _lt('Address') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'AddressFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const router = useRouter();

function onCountryValueClick(payload: ManyToOneValueClickPayload<Country>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CountryDetail', params: { id } });
}

function onStateValueClick(payload: ManyToOneValueClickPayload<State>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'StateDetail', params: { id } });
}

function onCityValueClick(payload: ManyToOneValueClickPayload<City>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CityDetail', params: { id } });
}
</script>

