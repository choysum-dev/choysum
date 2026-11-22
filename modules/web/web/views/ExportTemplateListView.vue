<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Admin ledger for web.ExportTemplate. End-user save/apply stays on ExportPanel;
    this List is sys.admin cross-model cleanup and technical CRUD.
  -->
  <ChoyListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    create-action="/web/export-templates/new"
    :action-ids="{
      create: exportTemplateActions.create,
      delete: exportTemplateActions.delete,
    }"
    :has-action="hasAction"
    detail-path="/web/export-templates/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end', width: 44 }" />
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
      :vColumnProps="{ minWidth: 140 }"
    />
    <ChoyVarcharField
      :store="store"
      prop="UserId"
      :label="_t('User')"
      :vColumnProps="{ minWidth: 120 }"
    />
    <ChoyBooleanField
      :store="store"
      prop="ImportCompatible"
      :label="_t('Import Compatible')"
      :vColumnProps="{ minWidth: 120 }"
    />
    <ChoyJsonField
      :store="store"
      prop="Fields"
      :label="_t('Fields')"
      :vColumnProps="{ minWidth: 180 }"
    />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyJsonField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
} from '@/web';
import { exportTemplateActions } from './export_template_actions';

defineOptions({ name: 'ExportTemplateListView', inheritAttrs: true });
const { _t } = createTranslate('web', { scope: 'web/views/ExportTemplateListView' });


const props = defineProps<{
  store?: ModelStore<any>;
}>();

const store = resolvePageStore(props.store, 'ExportTemplateListView');
const { hasAction } = usePermission();


const { listRef, expose } = useListViewExpose<any>();
defineExpose(expose);
</script>
