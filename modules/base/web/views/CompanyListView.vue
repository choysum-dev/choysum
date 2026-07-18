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
    :action-ids="{ create: companyActions.create, delete: companyActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <OVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <OVColumn type="index" :vColumnProps="{ align: 'right' }" />

    <OVarCharField :store="store" prop="Name" :label="_t('Name')" />
    <OVarCharField :store="store" prop="Code" :label="_t('Code')" />
    <OManyToOneField :store="store" prop="ParentId" :label="_t('Parent Company')">
      <OVarCharField :store="store" prop="ParentId.Name" :label="_t('Name')" />
    </OManyToOneField>
    <ODateTimeField :store="store" prop="CreatedAt" :label="_t('Created At')" />
    <ODateTimeField :store="store" prop="UpdatedAt" :label="_t('Updated At')" />
  </OListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type Company from '@/base/service/models/company';
import { useRouter } from 'vue-router';

import OListView from '@/web/web/components/view/OListView.vue';
import type { RowEventPayload } from '@/web/web/components/view/listViewTypes';
import OVColumn from '@/web/web/components/vtable/OVColumn.vue';
import OVarCharField from '@/web/web/components/field/OVarCharField.vue';
import ODateTimeField from '@/web/web/components/field/ODatetimeField.vue';
import OManyToOneField from '@/web/web/components/field/OManyToOneField.vue';
import OSearchView from '@/web/web/components/view/OSearchView.vue';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'CompanyListView', inheritAttrs: true });
const { _t } = createTranslate('base', { scope: 'web/views/CompanyListView' });
const { _t: _tRef } = createTranslate('base', { output: 'reference', scope: 'web/views/CompanyListView' });

const props = defineProps<{
  store: WebModelStore<Company>;
}>();

const companyActions = defineModelActions('base.Company', { entityTitle: _tRef('Company') });
const { hasAction } = usePermission();
const router = useRouter();

function onRowClick(payload: RowEventPayload<Company>) {
  router.push(`/base/companies/${payload.row.Id}`);
}

const { listRef, expose } = useListViewExpose<Company>();
defineExpose(expose);
</script>
