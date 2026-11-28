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
    :detail-path="detailPathForRow"
  >
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end' }" />
    <ChoyVarcharField
      :store="store"
      prop="Module"
      :label="_t('Module')"
    />
    <ChoyVarcharField
      :store="store"
      prop="Name"
      :label="_t('Name')"
    />
    <ChoyVarcharField
      :store="store"
      prop="Application"
      :label="_t('Application')"
    />
    <ChoyVarcharField
      :store="store"
      prop="ModelName"
      :label="_t('Model Name')"
    />
    <ChoyVarcharField
      :store="store"
      prop="ResId"
      :label="_t('Resource Id')"
    />
    <ChoyBooleanField
      :store="store"
      prop="NoUpdate"
      :label="_t('No Update')"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
} from '@/web';

defineOptions({ name: 'MetaModelDataListView', inheritAttrs: true });
const { _t } = createTranslate('meta', { scope: 'web/views/MetaModelDataListView' });

const props = defineProps<{
  store?: ModelStore<any>;
}>();

const store = resolvePageStore(props.store, 'MetaModelDataListView');

function detailPathForRow(row: Record<string, unknown>) {
  const id = String(row?.Id ?? '').trim();
  if (!id) return null;
  return { name: 'MetaModelDataDetail', params: { id } };
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
