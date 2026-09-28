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
    :action-ids="{ create: languageActions.create, delete: languageActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoySelectionField :store="store" prop="Direction" />
    <ChoyVarcharField :store="store" prop="DecimalSeparator" />
    <ChoyVarcharField :store="store" prop="ThousandSeparator" />
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Language from '@/base/service/models/language';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';
import { ChoyBooleanField, ChoyListView, ChoySearchView, ChoySelectionField, ChoyVColumn, ChoyVarcharField } from '@/web';

defineOptions({ name: 'LanguageListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/LanguageListView' });
const props = defineProps<{ store?: WebModelStore<Language> }>();
const store = resolvePageStore(props.store, 'LanguageListView');
const languageActions = defineModelActions('base.Language', { entityTitle: _lt('Language') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/base/languages/${id}`);
}
const { listRef, expose } = useListViewExpose<Language>();
defineExpose(expose);
</script>
