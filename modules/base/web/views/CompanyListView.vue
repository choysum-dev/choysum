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
    :action-ids="{ create: companyActions.create, delete: companyActions.delete }"
    :has-action="hasAction"
    detail-path="/base/companies/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />

    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyManyToOneField :store="store" prop="ParentId">
      <ChoyVarcharField :store="store" prop="ParentId.Name" :label="_t('Name')" />
    </ChoyManyToOneField>
    <ChoyDatetimeField :store="store" prop="CreatedAt" />
    <ChoyDatetimeField :store="store" prop="UpdatedAt" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type Company from '@/base/service/models/company';

import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoyTableColumn, ChoyVarcharField} from '@/web';

defineOptions({ name: 'CompanyListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/CompanyListView' });

const props = defineProps<{
  store?: ModelStore<Company>;
}>();
const store = resolvePageStore(props.store, 'CompanyListView');

const companyActions = defineModelActions('base.Company', { entityTitle: _lt('Company') });
const { hasAction } = usePermission();


const { listRef, expose } = useListViewExpose<Company>();
defineExpose(expose);
</script>
