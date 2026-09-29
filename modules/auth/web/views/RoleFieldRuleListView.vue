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
    :show-header="showHeader"
    :action-ids="{ create: fieldRuleActions.create, delete: fieldRuleActions.delete }"
    :has-action="hasAction"
    @row-click="onRowClick"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="RoleId.Name" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyManyToOneRefField prop="MetaApplicationId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoyManyToOneRefField prop="MetaModelId" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyManyToOneRefField prop="MetaFieldId" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoySelectionField prop="LogicalModelName" :store="store" :vColumnProps="{ minWidth: 140 }" />
    <ChoySelectionField prop="PermRead" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoySelectionField prop="PermWrite" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type RoleFieldRule from '@/auth/service/models/role_field_rule';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoySelectionField, ChoyTableColumn, ChoyVarcharField, ChoyManyToOneRefField} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { resolveListRowRecordId } from './list_row_nav';

defineOptions({ name: 'RoleFieldRuleListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleFieldRuleListView' });


const router = useRouter();

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<RoleFieldRule>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleFieldRuleListView');
const { showHeader } = props;
const fieldRuleActions = defineModelActions('auth.RoleFieldRule', { entityTitle: _lt('Field Rule') });
const { hasAction } = usePermission();

/**
 * Open the clicked field-rule row in detail view.
 */
function onRowClick(row: Record<string, unknown>) {
  const id = resolveListRowRecordId(row);
  if (id) router.push(`/auth/field-rules/${id}`);
}

const { listRef, expose } = useListViewExpose<RoleFieldRule>();
defineExpose(expose);
</script>
