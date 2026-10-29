<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    class="choy-properties-definition-selection w-full min-w-0 overflow-hidden rounded-md border border-border"
    data-testid="choy-properties-definition-selection"
  >
    <div class="w-full overflow-x-auto">
      <table class="w-full border-collapse text-sm">
        <thead>
          <tr class="border-b border-border bg-muted/40 text-left text-muted-foreground">
            <th v-if="!readonly" class="w-9 px-1 py-1.5 font-medium" />
            <th class="whitespace-nowrap px-3 py-1.5 font-medium">{{ _t('Value') }}</th>
            <th class="min-w-[10rem] px-3 py-1.5 font-medium">{{ _t('Label') }}</th>
            <th class="w-10 px-2 py-1.5 font-medium" />
          </tr>
        </thead>
        <tbody>
          <tr v-if="rows.length === 0" data-testid="choy-properties-definition-selection-empty">
            <td class="px-3 py-3 text-sm text-muted-foreground" :colspan="readonly ? 3 : 4">
              {{ _t('No options') }}
            </td>
          </tr>
          <tr
            v-for="(row, index) in rows"
            :key="rowKey(row, index)"
            class="border-b border-border last:border-b-0"
            :class="selectionRowDragClass(index)"
            data-testid="choy-properties-definition-selection-row"
            :data-index="index"
            @dragover="!readonly && onRowDragOver(index, $event)"
            @drop="!readonly && onRowDrop(index, $event)"
          >
            <td v-if="!readonly" class="w-9 px-1 py-1 align-middle">
              <span
                class="inline-flex size-7 cursor-grab items-center justify-center text-muted-foreground select-none active:cursor-grabbing"
                draggable="true"
                :title="_t('Reorder')"
                :aria-label="_t('Reorder')"
                data-testid="choy-properties-definition-selection-handle"
                @dragstart="onHandleDragStart(index, $event)"
                @dragend="handleReorder.onDragEnd()"
                @click.stop
              >
                <GripVertical class="size-3.5" aria-hidden="true" />
              </span>
            </td>
            <td class="px-2 py-1 align-middle">
              <Input
                :model-value="row.value"
                class="h-8 w-full min-w-[6rem] font-mono text-xs"
                :disabled="readonly"
                data-testid="choy-properties-definition-selection-value"
                @update:model-value="onValue(index, $event)"
              />
            </td>
            <td class="px-2 py-1 align-middle">
              <div class="flex min-w-0 items-center gap-1">
                <Input
                  :model-value="row.label"
                  class="h-8 min-w-0 flex-1"
                  :disabled="readonly"
                  data-testid="choy-properties-definition-selection-label"
                  @update:model-value="onLabel(index, $event)"
                />
                <ChoyButton
                  v-if="!readonly"
                  type="button"
                  size="icon"
                  variant="ghost"
                  class="size-8 shrink-0 text-muted-foreground hover:text-foreground"
                  :disabled="!String(row.value || '').trim()"
                  :aria-label="_t('Translate option')"
                  :title="_t('Translate option')"
                  data-testid="choy-properties-definition-selection-translate"
                  @click="emit('translate', String(row.value || '').trim())"
                >
                  <Languages class="size-4" aria-hidden="true" />
                </ChoyButton>
              </div>
            </td>
            <td class="px-1 py-1 align-middle">
              <ChoyButton
                v-if="!readonly"
                type="button"
                size="icon"
                variant="ghost"
                class="size-7 text-muted-foreground hover:text-destructive"
                :aria-label="_t('Delete')"
                data-testid="choy-properties-definition-selection-remove"
                @click="onRemove(index)"
              >
                <Trash2 class="size-3.5" aria-hidden="true" />
              </ChoyButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div
      v-if="!readonly"
      class="flex items-center border-t border-border px-3 py-2"
      data-testid="choy-properties-definition-selection-add-footer"
    >
      <ChoyButton
        type="button"
        size="sm"
        variant="link"
        class="h-auto px-0"
        data-testid="choy-properties-definition-selection-add"
        @click="onAdd"
      >
        {{ _t('Add a line…') }}
      </ChoyButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { GripVertical, Languages, Trash2 } from 'lucide-vue-next';
import { createTranslate } from '@/web/web/i18n';
import {
  setListHandleDragImage,
  useListHandleReorder,
} from '@/web/web/composables/useListHandleReorder';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import Input from '../vendor/ui/input/Input.vue';
import type { SelectionDraftRow } from './propertiesDefinitionHelpers';

defineOptions({ name: 'ChoyPropertiesDefinitionSelectionEditor' });

const { _t } = createTranslate('web', {
  scope: 'web/components/field/ChoyPropertiesDefinitionSelectionEditor',
});

const props = withDefaults(
  defineProps<{
    modelValue?: SelectionDraftRow[] | null;
    readonly?: boolean;
  }>(),
  {
    modelValue: () => [],
    readonly: false,
  },
);

const emit = defineEmits<{
  'update:modelValue': [rows: SelectionDraftRow[]];
  translate: [value: string];
}>();

const rows = computed(() => (Array.isArray(props.modelValue) ? props.modelValue : []));
const handleEnabled = computed(() => !props.readonly);

const handleReorder = useListHandleReorder({
  rows: () => rows.value as SelectionDraftRow[],
  enabled: handleEnabled,
  getRecord: () => ({}),
  onReorder: next => emitRows(next as SelectionDraftRow[]),
});

function rowKey(_row: SelectionDraftRow, index: number): string {
  // Index-only: value is editable; including it remounts inputs on every keystroke.
  return `row-${index}`;
}

function emitRows(next: SelectionDraftRow[]) {
  emit(
    'update:modelValue',
    next.map(r => ({ value: String(r.value ?? ''), label: String(r.label ?? '') })),
  );
}

function onValue(index: number, raw: string | number) {
  const next = rows.value.map((r, i) => (i === index ? { ...r, value: String(raw ?? '') } : r));
  emitRows(next);
}

function onLabel(index: number, raw: string | number) {
  const next = rows.value.map((r, i) => (i === index ? { ...r, label: String(raw ?? '') } : r));
  emitRows(next);
}

function onRemove(index: number) {
  emitRows(rows.value.filter((_, i) => i !== index));
}

function onAdd() {
  emitRows([...rows.value, { value: '', label: '' }]);
}

function selectionRowDragClass(index: number): Record<string, boolean> {
  if (props.readonly) return {};
  const dragging = handleReorder.draggingIndex.value;
  const over = handleReorder.overIndex.value;
  return {
    'opacity-60': dragging === index,
    'bg-accent/40 ring-1 ring-inset ring-primary/30':
      dragging != null && over === index && over !== dragging,
  };
}

function onHandleDragStart(index: number, event: DragEvent) {
  event.stopPropagation();
  handleReorder.onDragStart(index, event);
  setListHandleDragImage(event);
}

function onRowDragOver(index: number, event: DragEvent) {
  event.preventDefault();
  handleReorder.onDragOver(index, event);
}

function onRowDrop(index: number, event: DragEvent) {
  event.preventDefault();
  void handleReorder.onDrop(index, event);
}

defineExpose({ handleReorder, selectionRowDragClass, onHandleDragStart, onRowDragOver, onRowDrop });
</script>
