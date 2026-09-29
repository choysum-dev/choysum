<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFormView
    v-bind="{ store, recordId, initialValues, viewMode, showHeader, createAction }"
    :action-ids="{ create: companyActions.create, edit: companyActions.edit, copy: companyActions.copy, delete: companyActions.delete }"
    :has-action="hasAction"
    v-on="$attrs"
  >
    <ChoyCard :title="_t('Basic Information')" class="mb-3.5"><ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="Name" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyVarcharField :store="store" prop="Code" :rules="requiredRules" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoySelectionField
            :store="store"
            prop="Timezone"
            :rules="requiredRules"
            :placeholder="_t('Select a time zone')"
            :select-props="{ filterable: true, allowCreate: false }"
          />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyManyToOneField
            :store="store"
            prop="CurrencyId"
            :rules="requiredRules"
            :search-view="CurrencyListView"
            :search-view-title="_t('Select Currency')"
            @value-click="onCurrencyValueClick"
          />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyManyToOneField
            :store="store"
            prop="ParentId"
            :search-view="CompanyListView"
            :search-view-title="_t('Select Parent Company')"
            @value-click="onParentCompanyValueClick"
          />
        </ChoyCol>
      </ChoyGrid>
      <ChoyGrid :cols="12">
        <ChoyCol :span="3">
          <ChoyDatetimeField :store="store" prop="CreatedAt" />
        </ChoyCol>
        <ChoyCol :span="3">
          <ChoyDatetimeField :store="store" prop="UpdatedAt" />
        </ChoyCol>
      </ChoyGrid>
    </ChoyCard>
  </ChoyFormView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { RouteLocationRaw } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Company from '@/base/service/models/company';
import type Currency from '@/base/service/models/currency';

import type { ValueClickPayload as ManyToOneValueClickPayload } from '@/web/web/components/field/manyToOneTypes';
import CompanyListView from './CompanyListView.vue';
import CurrencyListView from './CurrencyListView.vue';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import { ChoyCard, ChoyCol, ChoyDatetimeField, ChoyFormView, ChoyGrid, ChoyManyToOneField, ChoySelectionField, ChoyVarcharField} from '@/web';
import type { ChoyViewMode as ViewMode } from '@/web';

defineOptions({ name: 'CompanyFormView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CompanyFormView' });
const requiredRules = computed(() => [{ required: true, message: _t('Required') }]);

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<Company>;
    recordId?: string;
    initialValues?: Partial<Company>;
    viewMode?: ViewMode;
    showHeader?: boolean;
    createAction?: string | RouteLocationRaw;
  }>(),
  {
    showHeader: true,
    createAction: undefined,
  }
);

const companyActions = defineModelActions('base.Company', { entityTitle: _lt('Company') });
const { hasAction } = usePermission();
const store = resolvePageStore(props.store, 'CompanyFormView');
const { recordId, viewMode, showHeader, createAction } = props;
const { initialValues } = props;
const router = useRouter();

function onParentCompanyValueClick(payload: ManyToOneValueClickPayload<Company>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CompanyDetail', params: { id } });
}

function onCurrencyValueClick(payload: ManyToOneValueClickPayload<Currency>) {
  const id = String(payload?.id || '').trim();
  if (!id) return;
  void router.push({ name: 'CurrencyDetail', params: { id } });
}
</script>

