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
    :action-ids="{ create: uomCategoryActions.create, delete: uomCategoryActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Code" :label="_t('Code')" />
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type UoMCategory from '@/base/service/models/uom_category';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'UoMCategoryListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/UoMCategoryListView' });
const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/UoMCategoryListView' });
const props = defineProps<{ store: WebModelStore<UoMCategory> }>();
const uomCategoryActions = defineModelActions('base.UoMCategory', { entityTitle: _tRef('Unit of Measure Category') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<UoMCategory>) {
  router.push(`/base/uom-categories/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<UoMCategory>();
defineExpose(expose);
</script>
