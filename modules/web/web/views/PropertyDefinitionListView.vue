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
      create: propertyDefinitionActions.create,
      delete: propertyDefinitionActions.delete,
    }"
    :has-action="hasAction"
    :detail-path="detailPathForRow"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end' }" />
    <ChoyVarcharField
      :store="store"
      prop="TargetModel"
      :label="_t('Target Model')"
    />
    <ChoyVarcharField
      :store="store"
      prop="PropertiesField"
      :label="_t('Properties Field')"
    />
    <ChoyVarcharField
      :store="store"
      prop="ContainerModel"
      :label="_t('Container Model')"
    />
    <ChoyVarcharField
      :store="store"
      prop="ContainerId"
      :label="_t('Container Id')"
    />
    <ChoyPropertiesDefinitionField
      :store="store"
      prop="Definition"
      :label="_t('Definition')"
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
  ChoyListView,
  ChoyPropertiesDefinitionField,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
} from '@/web';
import { propertyDefinitionActions } from './property_definition_actions';

defineOptions({ name: 'PropertyDefinitionListView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/PropertyDefinitionListView' });

const props = defineProps<{
  store?: ModelStore<any>;
  application: string;
}>();

const application = computed(() => String(props.application || '').trim());
const store = resolvePageStore(props.store, 'PropertyDefinitionListView');
const createPath = computed(() => {
  const app = application.value;
  return app ? `/web/property-definitions/${app}/new` : '';
});

const { hasAction } = usePermission();

function detailPathForRow(row: Record<string, unknown>) {
  const id = String(row?.Id ?? '').trim();
  const app = application.value;
  if (!id || !app) return null;
  return `/web/property-definitions/${app}/${id}`;
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
