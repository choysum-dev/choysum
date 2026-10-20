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
    :action-ids="{ create: cityActions.create, delete: cityActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyManyToOneField :store="store" prop="CountryId"><ChoyVarcharField :store="store" prop="CountryId.Name" /></ChoyManyToOneField>
    <ChoyManyToOneField :store="store" prop="StateId"
      ><ChoyVarcharField :store="store" prop="StateId.Name"
    /></ChoyManyToOneField>
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type City from '@/base/service/models/city';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { ChoyBooleanField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoyTableColumn, ChoyVarcharField} from '@/web';
import type { RowEventPayload } from '@/web';

defineOptions({ name: 'CityListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CityListView' });
const props = defineProps<{ store?: WebModelStore<City> }>();
const store = resolvePageStore(props.store, 'CityListView');
const cityActions = defineModelActions('base.City', { entityTitle: _lt('City') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload) {
  const id = String(payload.row?.Id ?? '').trim();
  if (id) router.push(`/base/cities/${id}`);
}
const { listRef, expose } = useListViewExpose<City>();
defineExpose(expose);
</script>
