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
    detail-path="/auth/field-rules/:id"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField prop="RoleId.Name" :store="store" />
    <ChoyManyToOneRefField prop="MetaApplicationId" :store="store" />
    <ChoyManyToOneRefField prop="MetaModelId" :store="store" />
    <ChoyManyToOneRefField prop="MetaFieldId" :store="store" />
    <ChoySelectionField prop="LogicalModelName" :store="store" />
    <ChoySelectionField prop="PermRead" :store="store" />
    <ChoySelectionField prop="PermWrite" :store="store" />
    <ChoyDatetimeField prop="CreatedAt" mode="datetime" :store="store" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { ModelStore } from '@/web/web/stores/modelStore';
import type RoleFieldRule from '@/auth/service/models/role_field_rule';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { ChoyDatetimeField, ChoyListView, ChoyManyToOneField, ChoySearchView, ChoySelectionField, ChoyTableColumn, ChoyVarcharField, ChoyManyToOneRefField} from '@/web';
import { createTranslate } from '@/web/web/i18n';

defineOptions({ name: 'RoleFieldRuleListView', inheritAttrs: true });
const { _lt } = createTranslate('auth', { scope: 'web/views/RoleFieldRuleListView' });



const props = withDefaults(
  defineProps<{
    store?: ModelStore<RoleFieldRule>;
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

const { listRef, expose } = useListViewExpose<RoleFieldRule>();
defineExpose(expose);
</script>
