<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :action-ids="{ create: countryActions.create, delete: countryActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyVarcharField :store="store" prop="PhonePrefix" />
    <ChoyManyToOneField :store="store" prop="DefaultCurrencyId"
      ><ChoyVarcharField :store="store" prop="DefaultCurrencyId.Name" :label="_t('Currency')"
    /></ChoyManyToOneField>
    <ChoyBooleanField :store="store" prop="ZipRequired" />
    <ChoyBooleanField :store="store" prop="StateRequired" />
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Country from '@/base/service/models/country';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';
import { ChoyBooleanField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';

defineOptions({ name: 'CountryListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CountryListView' });
const props = defineProps<{ store?: WebModelStore<Country> }>();
const store = resolvePageStore(props.store, 'CountryListView');
const countryActions = defineModelActions('base.Country', { entityTitle: _lt('Country') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/base/countries/${id}`);
}
const { listRef, expose } = useListViewExpose<Country>();
defineExpose(expose);
</script>
