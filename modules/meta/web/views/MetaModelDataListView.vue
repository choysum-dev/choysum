<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Read-only admin ledger for meta.MetaModelData (Settings: External IDs).
    Loaders / import / uninstall own writes; this List is sys.admin diagnostics.
  -->
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    @row-click="onRowClick"
  >
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end', width: 44 }" />
    <ChoyVarcharField
      :store="store"
      prop="Module"
      :label="_t('Module')"
      :vColumnProps="{ minWidth: 120 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="Name"
      :label="_t('Name')"
      :vColumnProps="{ minWidth: 160 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="Application"
      :label="_t('Application')"
      :vColumnProps="{ minWidth: 120 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="ModelName"
      :label="_t('Model Name')"
      :vColumnProps="{ minWidth: 160 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="ResId"
      :label="_t('Resource Id')"
      :vColumnProps="{ minWidth: 140 }"
    />
    <ChoyBooleanField
      :store="store"
      prop="NoUpdate"
      :label="_t('No Update')"
      :vColumnProps="{ minWidth: 100 }"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
  type RowEventPayload,
} from '@/web';

defineOptions({ name: 'MetaModelDataListView', inheritAttrs: true });
const { _t } = createTranslate('meta', { scope: 'web/views/MetaModelDataListView' });

const router = useRouter();

const props = defineProps<{
  store?: WebModelStore<any>;
}>();

const store = resolvePageStore(props.store, 'MetaModelDataListView');

function onRowClick(payload: RowEventPayload) {
  const id = String(payload.row?.Id ?? '').trim();
  if (!id) return;
  router.push({ name: 'MetaModelDataDetail', params: { id } });
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
