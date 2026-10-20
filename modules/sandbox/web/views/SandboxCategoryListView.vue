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
    :action-ids="{ create: categoryActions.create, delete: categoryActions.delete }"
    :has-action="hasAction"
  >
    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'right' }" />
    <ChoyVarcharField :store="store" prop="Name" />
    <ChoyVarcharField :store="store" prop="Code" />
    <ChoyBooleanField :store="store" prop="IsActive" widget="checkbox" />
  </ChoyListView>
</template>

<script setup lang="ts">
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type SandboxCategory from '@/sandbox/service/models/sandbox_category';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { defineModelActions } from '@/core/web/resource';
import { usePermission } from '@/auth/web/composables/usePermission';
import { createTranslate } from '@/web/web/i18n';
import {
  ChoyBooleanField,
  ChoyListView,
  ChoySearchView,
  ChoyTableColumn,
  ChoyVarcharField,
} from '@/web';

defineOptions({ name: 'SandboxCategoryListView', inheritAttrs: true });
const { _lt } = createTranslate('sandbox', { scope: 'web/views/SandboxCategoryListView' });

const props = defineProps<{
  store?: WebModelStore<SandboxCategory>;
}>();
const store = resolvePageStore(props.store, 'SandboxCategoryListView');

const categoryActions = defineModelActions('sandbox.SandboxCategory', {
  entityTitle: _lt('Sandbox Category'),
});
const { hasAction } = usePermission();

const { listRef, expose } = useListViewExpose<SandboxCategory>();
defineExpose(expose);
</script>
