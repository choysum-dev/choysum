<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    data-anchor="choy.one-to-many-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="error"
    :name="name"
    :visible="visible"
  >
    <template #default>
      <div class="choy-one-to-many-field flex flex-col gap-2">
        <div v-if="!readonly && !disabled" class="flex flex-wrap gap-2">
          <ChoyButton size="sm" @click="onAdd">{{ addLabel }}</ChoyButton>
          <slot name="actions" />
        </div>

        <DataTable
          v-if="widget === 'list'"
          :columns="columns"
          :data="model"
          :row-id="rowKey"
          :height="height"
          :enable-row-selection="false"
          :enable-sorting="true"
          @row-click="onRowClick"
        />

        <div
          v-else
          class="grid gap-2"
          :style="{ gridTemplateColumns: 'repeat(auto-fill, minmax(10rem, 1fr))' }"
        >
          <div
            v-for="(row, index) in model"
            :key="String(rowKey(row, index))"
            class="rounded-md border border-border bg-background p-3 shadow-sm"
            @click="onCardClick(row)"
          >
            <div class="text-sm font-medium">{{ rowTitle(row) }}</div>
            <div v-if="rowSubtitle(row)" class="mt-1 text-xs text-foreground/60">
              {{ rowSubtitle(row) }}
            </div>
            <div v-if="removable && !readonly && !disabled" class="mt-2">
              <ChoyButton
                size="sm"
                variant="ghost"
                @click.stop="onRemove(row)"
              >
                Remove
              </ChoyButton>
            </div>
          </div>
          <div
            v-if="model.length === 0"
            class="col-span-full rounded-md border border-dashed border-border px-3 py-6 text-center text-xs text-foreground/50"
          >
            {{ emptyLabel }}
          </div>
        </div>

        <Dialog v-model:open="dialogOpen">
          <DialogContent>
            <DialogTitle>{{ dialogTitle }}</DialogTitle>
            <DialogDescription>One-to-many card detail (host may replace via slots).</DialogDescription>
            <slot name="dialog" :row="dialogRow">
              <pre class="max-h-60 overflow-auto rounded-md bg-muted/30 p-2 text-xs">{{
                dialogRow ? JSON.stringify(dialogRow, null, 2) : ''
              }}</pre>
            </slot>
          </DialogContent>
        </Dialog>
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends Record<string, unknown>">
import { computed, ref } from 'vue';
import type { ColumnDef } from '@tanstack/vue-table';
import DataTable from '../internal/DataTable.vue';
import type { DataTableRowId } from '../internal/dataTableHelpers';
import type { ClassValue } from '../../lib/utils';
import ChoyButton from '../layout/ChoyButton.vue';
import Dialog from '../vendor/ui/dialog/Dialog.vue';
import DialogContent from '../vendor/ui/dialog/DialogContent.vue';
import DialogDescription from '../vendor/ui/dialog/DialogDescription.vue';
import DialogTitle from '../vendor/ui/dialog/DialogTitle.vue';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

export type ChoyOneToManyWidget = 'list' | 'kanban';

/**
 * One-to-many relation field. list → DataTable; kanban → card grid + dialog.
 * Host owns the row array via defineModel (no useField / child store).
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      widget?: ChoyOneToManyWidget;
      columns?: ColumnDef<T, unknown>[];
      rowId?: (row: T) => DataTableRowId;
      titleField?: string;
      subtitleField?: string;
      height?: number;
      editable?: boolean;
      removable?: boolean;
      addLabel?: string;
      emptyLabel?: string;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    widget: 'list',
    columns: () => [],
    titleField: 'Title',
    height: 220,
    editable: true,
    removable: true,
    addLabel: 'Add',
    emptyLabel: 'No rows',
  },
);

const model = defineModel<T[]>({ default: () => [] });

const emit = defineEmits<{
  'add-click': [];
  'card-click': [row: T];
  'edit-request': [row: T];
  'remove-request': [row: T];
}>();

const dialogOpen = ref(false);
const dialogRow = ref<T | null>(null);

function rowKey(row: T, index = 0): DataTableRowId {
  if (props.rowId) return props.rowId(row);
  const id = (row as Record<string, unknown>).Id ?? (row as Record<string, unknown>).id;
  if (id != null && String(id).trim() !== '') return String(id);
  return `__o2m_${index}`;
}

function rowTitle(row: T): string {
  const v = (row as Record<string, unknown>)[props.titleField];
  return v == null || v === '' ? String(rowKey(row)) : String(v);
}

function rowSubtitle(row: T): string | undefined {
  if (!props.subtitleField) return undefined;
  const v = (row as Record<string, unknown>)[props.subtitleField];
  return v == null || v === '' ? undefined : String(v);
}

const dialogTitle = computed(() => (dialogRow.value ? rowTitle(dialogRow.value) : 'Row'));

let blankRowSeq = 0;

function canEdit(): boolean {
  return props.editable && !props.readonly && !props.disabled;
}

function onAdd(): void {
  if (canEdit() && props.widget === 'kanban') {
    // Kanban self-inserts; skip add-click so hosts do not append a second row.
    const generatedId = `new_${Date.now()}_${blankRowSeq++}`;
    const blank = {
      ...(props.titleField === 'Id' ? {} : { [props.titleField]: 'New row' }),
      Id: generatedId,
    } as unknown as T;
    model.value = [...model.value, blank];
    dialogRow.value = blank;
    dialogOpen.value = true;
    return;
  }
  emit('add-click');
}

function onCardClick(row: T): void {
  emit('card-click', row);
  if (!canEdit()) return;
  dialogRow.value = row;
  dialogOpen.value = true;
  emit('edit-request', row);
}

function onRemove(row: T): void {
  emit('remove-request', row);
  if (!props.removable || props.readonly || props.disabled) return;
  const index = model.value.indexOf(row);
  if (index < 0) return;
  model.value = model.value.filter((_, i) => i !== index);
}

function onRowClick(row: T): void {
  emit('card-click', row);
  if (canEdit()) emit('edit-request', row);
}
</script>
