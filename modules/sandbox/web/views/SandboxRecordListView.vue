<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyListView
    ref="listRef"
    class="sandbox-record-list-view"
    v-bind="$attrs"
    :store="store"
    :searchView="ChoySearchView"
    :show-header="showHeader"
    :action-ids="{ create: sandboxRecordActions.create, delete: sandboxRecordActions.delete }"
    :has-action="hasAction"
    detail-path="/sandbox/records/:id"
  >
    <template v-if="showHeader" #header-right>
      <ChoyViewSwitcher
        :aria-label="_t('View switcher')"
        :active="activeView"
        :options="viewSwitcherOptions"
        @select="onViewSelect"
      />
    </template>

    <ChoyTableColumn type="selection" :vColumnProps="{ align: 'center' }" />
    <ChoyTableColumn type="index" :vColumnProps="{ align: 'end' }" />

    <ChoyCharField prop="NameChar" :store="store" :vColumnProps="{ minWidth: 120 }" />
    <ChoyVarcharField prop="Title" :store="store" :vColumnProps="{ minWidth: 160 }" />
    <ChoyVarcharField prop="Code" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoySelectionField prop="State" :store="store" :vColumnProps="{ minWidth: 100 }" />
    <ChoyBooleanField :store="store" prop="IsActive" widget="checkbox" />
    <ChoyIntField prop="Sequence" :store="store" :vColumnProps="{ minWidth: 80 }" />
    <ChoyManyToOneField :store="store" prop="CategoryId">
      <ChoyVarcharField :store="store" prop="CategoryId.Name" :label="_t('Category')" />
    </ChoyManyToOneField>
    <ChoyDatetimeField prop="UpdatedAt" mode="datetime" :store="store" :vColumnProps="{ minWidth: 160 }" />
  </ChoyListView>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ModelStore } from '@/web/web/stores/modelStore';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import {
  ChoyBooleanField,
  ChoyCharField,
  ChoyDatetimeField,
  ChoyIntField,
  ChoyListView,
  ChoyManyToOneField,
  ChoySearchView,
  ChoySelectionField,
  ChoyTableColumn,
  ChoyVarcharField,
  ChoyViewSwitcher,
  useChoyViewRouteSwitch,
  type ChoyViewSwitcherOption,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { sandboxRecordActions } from './sandbox_record_actions';

defineOptions({ name: 'SandboxRecordListView', inheritAttrs: false });
const { _t } = createTranslate('sandbox', { scope: 'web/views/SandboxRecordListView' });


const props = withDefaults(
  defineProps<{
    store?: ModelStore<SandboxRecord>;
    showHeader?: boolean;
  }>(),
  {
    showHeader: true,
  },
);

const store = resolvePageStore(props.store, 'SandboxRecordListView');
const { active: activeView, selectView: onViewSelect } = useChoyViewRouteSwitch({ store });
const { showHeader } = props;
const { hasAction } = usePermission();

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'list', label: _t('List View') },
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'chart', label: _t('Chart View') },
]);


const { listRef, expose } = useListViewExpose<SandboxRecord>();
defineExpose(expose);
</script>
