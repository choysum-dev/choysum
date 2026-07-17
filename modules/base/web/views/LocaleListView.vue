<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OListView
    ref="listRef"
    v-bind="$attrs"
    :store="store"
    :searchView="OSearchView"
    :action-ids="{ create: localeActions.create, delete: localeActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Code" :label="_t('Code')" />
    <OVarCharField :store="store" prop="DateFormat" :label="_t('Date Format')" />
    <OVarCharField :store="store" prop="TimeFormat" :label="_t('Time Format')" />
    <OSelectionField :store="store" prop="CurrencySymbolPosition" :label="_t('Currency Symbol Position')" />
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Locale from '@/base/service/models/locale';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'LocaleListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/LocaleListView' });
const props = defineProps<{ store: WebModelStore<Locale> }>();
const localeActions = defineModelActions('base.Locale', { entityTitle: 'Locale' });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<Locale>) {
  router.push(`/base/locales/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<Locale>();
defineExpose(expose);
</script>
