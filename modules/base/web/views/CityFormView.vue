<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, viewMode, showHeader, createAction }"
    :action-ids="{ create: cityActions.create, edit: cityActions.edit, copy: cityActions.copy, delete: cityActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('City Information')" class="bfv-card"><ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" /></ChoyCol>
        <ChoyCol :span="4"><ChoyVarcharField :store="store" prop="Code" /></ChoyCol>
        <ChoyCol :span="4"><ChoyBooleanField :store="store" prop="IsActive" /></ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="CountryId"
            :search-view="CountryListView"
            :search-view-title="_t('Select Country')"
            @value-click="onCountryValueClick"
        /></ChoyCol>
        <ChoyCol :span="4"><ChoyManyToOneField
            :store="store"
            prop="StateId"
            :search-view="StateListView"
            :search-view-title="_t('Select State/Province')"
            @value-click="onStateValueClick"
        /></ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type City from '@/base/service/models/city';
import type Country from '@/base/service/models/country';
import type State from '@/base/service/models/state';
import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CountryListView from './CountryListView.vue';
import StateListView from './StateListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyCard, ChoyCol, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoyVarcharField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'CityFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CityFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);
const props = withDefaults(
  defineProps<{ store?: WebModelStore<City>; recordId?: string; viewMode?: ViewMode; showHeader?: boolean; createAction?: string | RouteLocationRaw }>(),
  { showHeader: true, createAction: undefined }
);
const cityActions = defineModelActions('base.City', { entityTitle: _lt('City') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'CityFormView');
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
</script>

<style scoped>
.bfv-card {
  margin-bottom: 14px;
}
</style>
