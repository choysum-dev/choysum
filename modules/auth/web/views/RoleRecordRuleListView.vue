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
    :action-ids="{ create: recordRuleActions.create, delete: recordRuleActions.delete }"
    :has-action="hasAction"
    detail-path="/auth/record-rules/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="RoleId.Name" :store="store" />
    <ChoySelectionField prop="Kind" :store="store" />
    <ChoyManyToOneRefField prop="MetaApplicationId" :store="store" />
    <ChoyManyToOneRefField prop="MetaModelId" :store="store" />
    <ChoyBooleanField prop="PermRead" :store="store" />
    <ChoyBooleanField prop="PermWrite" :store="store" />
    <ChoyBooleanField prop="PermCreate" :store="store" />
    <ChoyBooleanField prop="PermDelete" :store="store" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type RoleRecordRule from '@/auth/service/models/role_record_rule';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyBooleanField, ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoySelectionField, ChoyTableColumn, ChoyVarcharField, ChoyManyToOneRefField} from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleRecordRuleListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleRecordRuleListView' });



const props = withDefaults(
  defineProps<{
    store?: ModelStore<RoleRecordRule>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  }
);

const store = resolvePageStore(props.store, 'RoleRecordRuleListView');
const { showHeader } = props;
const recordRuleActions = defineModelActions('auth.RoleRecordRule', { entityTitle: _lt('Record Rule') });
const { hasAction } = usePermission();

/**
 * Open the clicked record-rule row in detail view.
 */

const { listRef, expose } = useListViewExpose<RoleRecordRule>();
defineExpose(expose);
</script>
