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

    <!-- Domain bulk workflow → List selection contextual (pairs with Form statusbar). -->
    <template #contextual-actions="{ selectedItems }">
      <ChoyActionButton
        v-if="canWriteRecord && selectedIdsFor('draft', selectedItems).length"
        data-testid="sandbox-list-mark-open"
        size="sm"
        variant="outline"
        :busy="bulkBusy"
        @execute="() => void markSelected(selectedItems, 'open')"
      >
        <ChevronRight class="size-4" aria-hidden="true" />
        {{ _t('Mark Open') }}
      </ChoyActionButton>
      <ChoyActionButton
        v-if="canWriteRecord && selectedIdsFor('open', selectedItems).length"
        data-testid="sandbox-list-mark-done"
        size="sm"
        variant="outline"
        :busy="bulkBusy"
        @execute="() => void markSelected(selectedItems, 'done')"
      >
        <CircleCheck class="size-4" aria-hidden="true" />
        {{ _t('Mark Done') }}
      </ChoyActionButton>
      <ChoyActionButton
        v-if="canWriteRecord && selectedIdsFor('done', selectedItems).length"
        data-testid="sandbox-list-reset-draft"
        size="sm"
        variant="outline"
        :busy="bulkBusy"
        @execute="() => void markSelected(selectedItems, 'draft')"
      >
        <RotateCcw class="size-4" aria-hidden="true" />
        {{ _t('Reset to Draft') }}
      </ChoyActionButton>
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
import { ChevronRight, CircleCheck, RotateCcw } from 'lucide-vue-next';
import type { ModelStore } from '@/web/web/stores/modelStore';
import type SandboxRecord from '@/sandbox/service/models/sandbox_record';
import { useListViewExpose } from '@/web/web/composables/useListView';
import { resolvePageStore } from '@/web/web/composables/usePageContext';
import { usePermission } from '@/auth/web/composables/usePermission';
import {
  ChoyActionButton,
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
  useListWorkflowAction,
  type ChoyViewSwitcherOption,
} from '@/web';
import { createTranslate } from '@/web/web/i18n';
import { sandboxRecordActions } from './sandbox_record_actions';
import {
  sandboxWorkflowSourceState,
  selectedSandboxIdsFor,
  type SandboxWorkflowState,
} from './sandbox_record_list_workflow';

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
const canWriteRecord = computed(() => hasAction(sandboxRecordActions.edit));

const viewSwitcherOptions = computed<ChoyViewSwitcherOption[]>(() => [
  { value: 'list', label: _t('List View') },
  { value: 'kanban', label: _t('Kanban View') },
  { value: 'chart', label: _t('Chart View') },
]);

const { listRef, expose } = useListViewExpose<SandboxRecord>();
defineExpose(expose);

const { busy: bulkBusy, run: runBulkWorkflow } = useListWorkflowAction({
  listRef: listRef as any,
  reload: 'apply',
  clearSelection: true,
  errorMessage: err =>
    err instanceof Error ? err.message : String(err || _t('Failed to update state')),
});

function selectedIdsFor(from: SandboxWorkflowState, selectedItems: unknown): string[] {
  return selectedSandboxIdsFor(from, selectedItems);
}

async function markSelected(selectedItems: unknown, next: SandboxWorkflowState): Promise<void> {
  const ids = selectedSandboxIdsFor(sandboxWorkflowSourceState(next), selectedItems);
  if (!ids.length) return;
  await runBulkWorkflow(async () => {
    for (const id of ids) {
      // UpdateById does not run @Onchange; mirror Progress like Form statusbar.
      await store.UpdateById(id, { State: next, Progress: next } as Partial<SandboxRecord>);
    }
  });
}
</script>
