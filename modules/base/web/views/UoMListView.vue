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
    :action-ids="{ create: uomActions.create, delete: uomActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Symbol" />
    <ChoyManyToOneField :store="store" prop="CategoryId"
      ><ChoyVarcharField :store="store" prop="CategoryId.Name"
    /></ChoyManyToOneField>
    <ChoyBooleanField :store="store" prop="IsReference" />
    <ChoyNumberField :store="store" prop="Factor" />
    <ChoyNumberField :store="store" prop="Rounding" />
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type UoM from '@/base/service/models/uom';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';
import { ChoyBooleanField, ChoyListView, ChoyManyToOneField, ChoyNumberField, ChoySearchView, ChoyVColumn, ChoyVarcharField } from '@/web';

defineOptions({ name: 'UoMListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/UoMListView' });
const props = defineProps<{ store?: WebModelStore<UoM> }>();
const store = resolvePageStore(props.store, 'UoMListView');
const uomActions = defineModelActions('base.UoM', { entityTitle: _lt('Unit of Measure') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/base/uoms/${id}`);
}
const { listRef, expose } = useListViewExpose<UoM>();
defineExpose(expose);
</script>
