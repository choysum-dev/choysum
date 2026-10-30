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
    @row-click="onRowClick"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end' }" />
    <ChoyVarcharField
      :store="store"
      prop="TargetModel"
      :label="_t('Target Model')"
      :vColumnProps="{ minWidth: 140 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="PropertiesField"
      :label="_t('Properties Field')"
      :vColumnProps="{ minWidth: 140 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="ContainerModel"
      :label="_t('Container Model')"
      :vColumnProps="{ minWidth: 140 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="ContainerId"
      :label="_t('Container Id')"
      :vColumnProps="{ minWidth: 120 }"
    />
    <ChoyPropertiesDefinitionField
      :store="store"
      prop="Definition"
      :label="_t('Definition')"
      :vColumnProps="{ minWidth: 180 }"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
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
  type RowEventPayload,
} from '@/web';
import { propertyDefinitionActions } from './property_definition_actions';

defineOptions({ name: 'PropertyDefinitionListView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/PropertyDefinitionListView' });

const router = useRouter();

const props = defineProps<{
  store?: WebModelStore<any>;
  application: string;
}>();

const application = computed(() => String(props.application || '').trim());
const store = resolvePageStore(props.store, 'PropertyDefinitionListView');
const createPath = computed(() => {
  const app = application.value;
  return app ? `/web/property-definitions/${app}/new` : '';
});

const { hasAction } = usePermission();

function onRowClick(payload: RowEventPayload) {
  const id = String(payload.row?.Id ?? '').trim();
  const app = application.value;
  if (id && app) router.push(`/web/property-definitions/${app}/${id}`);
}

const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
