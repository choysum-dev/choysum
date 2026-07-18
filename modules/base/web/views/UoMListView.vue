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
    :action-ids="{ create: uomActions.create, delete: uomActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Symbol" :label="_t('Symbol')" />
    <OManyToOneField :store="store" prop="CategoryId" :label="_t('Category')"
      ><OVarCharField :store="store" prop="CategoryId.Name" :label="_t('Category')"
    /></OManyToOneField>
    <OBooleanField :store="store" prop="IsReference" :label="_t('Reference Unit')" />
    <ODecimalField :store="store" prop="Factor" :label="_t('Conversion Factor')" />
    <ODecimalField :store="store" prop="Rounding" :label="_t('Rounding')" />
    <OBooleanField :store="store" prop="IsActive" :label="_t('Active')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type UoM from '@/base/service/models/uom';
import { useRouter } from 'vue-router';
import OListView, { type RowEventPayload } from '@/web/web/components/view/OListView.vue';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import OBooleanField from '@/web/web/components/field/OBooleanField.vue';
import ODecimalField from '@/web/web/components/field/ODecimalField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'UoMListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/UoMListView' });
const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/UoMListView' });
const props = defineProps<{ store: WebModelStore<UoM> }>();
const uomActions = defineModelActions('base.UoM', { entityTitle: _tRef('Unit of Measure') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(payload: RowEventPayload<UoM>) {
  router.push(`/base/uoms/${payload.row.Id}`);
}
const { listRef, expose } = useListViewExpose<UoM>();
defineExpose(expose);
</script>
