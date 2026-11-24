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
    :keyword-fields="keywordFields"
    :action-ids="{}"
    :detail-path="detailPathForRow"
  >
    <template #user-actions>
      <TerminologyExportMenu :application="application" />
    </template>
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Module" />
    <ChoyVarcharField :store="store" prop="Lang" />
    <ChoyVarcharField :store="store" prop="Scope" />
    <ChoyTextField :store="store" prop="Src" />
    <ChoyTextField :store="store" prop="Value" />
    <ChoyVarcharField :store="store" prop="Kind" />
    <ChoyVarcharField :store="store" prop="Source" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import { computed } from 'vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import {
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyTextField,
  ChoyVarcharField,
} from '@/web';
import TerminologyExportMenu from '../components/terminology/TerminologyExportMenu.vue';

defineOptions({ name: 'TranslationTermListView', inheritAttrs: true });

const props = defineProps<{
  store?: ModelStore<any>;
  application: string;
}>();

const application = computed(() => String(props.application || '').trim());
const store = resolvePageStore(props.store, 'TranslationTermListView');

/** Quick-search Src (source) and Value (translation); text fields are not in the char/varchar fallback. */
const keywordFields = ['Src', 'Value'];

function detailPathForRow(row: Record<string, unknown>) {
  const id = String(row?.Id ?? '').trim();
  const app = application.value;
  if (!id || !app) return null;
  return `/web/terminology/${app}/${id}`;
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
