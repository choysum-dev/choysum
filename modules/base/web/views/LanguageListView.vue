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
    :action-ids="{ create: languageActions.create, delete: languageActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" />
    <OVarCharField :store="store" prop="Code" />
    <OSelectionField :store="store" prop="Direction" />
    <OVarCharField :store="store" prop="DecimalSeparator" />
    <OVarCharField :store="store" prop="ThousandSeparator" />
    <OBooleanField :store="store" prop="IsActive" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Language from '@/base/service/models/language';
import { useRouter } from 'vue-router';
import OListView from '@/web/web/components/view/OListView.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OSelectionField from '@/web/web/components/field/OSelectionField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'LanguageListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/LanguageListView' });
const props = defineProps<{ store?: WebModelStore<Language> }>();
const store = resolvePageStore(props.store, 'LanguageListView');
const languageActions = defineModelActions('base.Language', { entityTitle: _lt('Language') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<Language>) {
  router.push(`/base/languages/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<Language>();
defineExpose(expose);
</script>
