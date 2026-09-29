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
    :action-ids="{ create: uomCategoryActions.create, delete: uomCategoryActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyVColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyVColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyBooleanField :store="store" prop="IsActive" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type UoMCategory from '@/base/service/models/uom_category';
import { useRouter } from 'vue-router';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';
import { ChoyBooleanField, ChoyListView, ChoySearchView, ChoyVColumn, ChoyVarcharField} from '@/web';

defineOptions({ name: 'UoMCategoryListView', inheritAttrs: true });
const { _t, _lt } = createTranslate('base', { scope: 'web/views/UoMCategoryListView' });
const props = defineProps<{ store?: WebModelStore<UoMCategory> }>();
const store = resolvePageStore(props.store, 'UoMCategoryListView');
const uomCategoryActions = defineModelActions('base.UoMCategory', { entityTitle: _lt('Unit of Measure Category') });
const { hasAction } = usePermission();
const router = useRouter();
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/base/uom-categories/${id}`);
}
const { listRef, expose } = useListViewExpose<UoMCategory>();
defineExpose(expose);
</script>
