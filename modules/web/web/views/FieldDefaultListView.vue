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
    :create-action="createPath"
    :action-ids="{
      create: fieldDefaultActions.create,
      delete: fieldDefaultActions.delete,
    }"
    :has-action="hasAction"
    :detail-path="detailPathForRow"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end' }" />
    <ChoyVarcharField
      :store="store"
      prop="Model"
      :label="_t('Model')"
    />
    <ChoyVarcharField
      :store="store"
      prop="Field"
      :label="_t('Field')"
    />
    <ChoyVarcharField
      :store="store"
      prop="UserId"
      :label="_t('User Id')"
    />
    <ChoyVarcharField
      :store="store"
      prop="CompanyId"
      :label="_t('Company Id')"
    />
    <ChoyJsonField
      :store="store"
      prop="Value"
      :label="_t('Value')"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ModelStore } from '@/web/web/stores/modelStore';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyJsonField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
} from '@/web';
import { fieldDefaultActions } from './field_default_actions';

defineOptions({ name: 'FieldDefaultListView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/FieldDefaultListView' });

const props = defineProps<{
  store?: ModelStore<any>;
  application: string;
}>();

const application = computed(() => String(props.application || '').trim());
const store = resolvePageStore(props.store, 'FieldDefaultListView');
const createPath = computed(() => {
  const app = application.value;
  return app ? `/web/field-defaults/${app}/new` : '';
});

const { hasAction } = usePermission();

function detailPathForRow(row: Record<string, unknown>) {
  const id = String(row?.Id ?? '').trim();
  const app = application.value;
  if (!id || !app) return null;
  return `/web/field-defaults/${app}/${id}`;
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
