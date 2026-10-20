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
    :action-ids="{ create: tagActions.create, delete: tagActions.delete }"
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
import type SandboxTag from '@/sandbox/service/models/sandbox_tag';
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

defineOptions({ name: 'SandboxTagListView', inheritAttrs: true });
const { _lt } = createTranslate('sandbox', { scope: 'web/views/SandboxTagListView' });

const props = defineProps<{
  store?: WebModelStore<SandboxTag>;
}>();
const store = resolvePageStore(props.store, 'SandboxTagListView');

const tagActions = defineModelActions('sandbox.SandboxTag', {
  entityTitle: _lt('Sandbox Tag'),
});
const { hasAction } = usePermission();

const { listRef, expose } = useListViewExpose<SandboxTag>();
defineExpose(expose);
</script>
