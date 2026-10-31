<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :orientation="orientation"
    :vColumnProps="vColumnProps"
    :toView="toView"
    :fromView="fromView"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    :hide-label="hideLabel"
    :label-for-control="false"
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue, inputId, labelId, describedBy, invalid }">
      <span
        v-if="isTableLike"
        class="truncate text-sm text-foreground"
        data-testid="choy-properties-definition-summary"
      >{{ summaryText(fieldValue().value) }}</span>
      <div
        v-else
        class="flex w-full min-w-0 flex-col gap-2"
        data-testid="choy-properties-definition-form"
        :id="inputId || undefined"
        :aria-labelledby="labelId || undefined"
        :aria-describedby="describedBy || undefined"
        :aria-invalid="invalid ? 'true' : undefined"
      >
        <p
          v-if="writeError"
          class="m-0 text-sm text-destructive"
          data-testid="choy-properties-definition-write-error"
        >
          {{ writeError }}
        </p>
        <DefinitionLinesTable
          :drafts="drafts"
          :can-mutate="canMutate"
          :empty-text="_t('No properties defined')"
          :name-header="_t('Name')"
          :type-header="_t('Type')"
          :label-header="_t('Label')"
          :remove-text="_t('Remove')"
          :reorder-text="_t('Reorder')"
          :index-header="'#'"
          :add-text="_t('Add a line…')"
          @open="openRow"
          @remove="onRemove"
          @reorder="onReorder"
          @add="onAdd"
        />
      </div>
    </template>

    <template #display="{ fieldValue }">
      <span
        v-if="isTableLike"
        class="truncate text-sm text-foreground"
        data-testid="choy-properties-definition-summary"
      >{{ summaryText(fieldValue().value) }}</span>
      <div v-else class="flex w-full min-w-0 flex-col gap-2" data-testid="choy-properties-definition-display">
        <DefinitionLinesTable
          :drafts="drafts"
          :can-mutate="false"
          :empty-text="_t('No properties defined')"
          :name-header="_t('Name')"
          :type-header="_t('Type')"
          :label-header="_t('Label')"
          :remove-text="_t('Remove')"
          :reorder-text="_t('Reorder')"
          :index-header="'#'"
          @open="openRow"
          @remove="onRemove"
          @reorder="onReorder"
        />
      </div>
    </template>
  </ChoyFieldBase>

  <ChoyPropertiesDefinitionItemDialog
    v-model="dialogOpen"
    :item="dialogItem"
    :readonly="dialogReadonly"
    :can-edit="canMutate"
    :commit-error="writeError"
    @saved="onDialogSaved"
    @edit="onDialogEdit"
  />
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, unknown[] | null | undefined>, V = FieldPathType<T, P>">
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { PropertyItemDefinition } from '@/core/service/orm/model/properties_types';
import { TRANSLATED_BASE_LANG } from '@/core/service/orm/model/properties_label_i18n';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { RuleItem } from 'async-validator';
import { computed, defineComponent, h, nextTick, ref, watch, type WritableComputedRef } from 'vue';
import { getActivePinia } from 'pinia';
import { GripVertical, Trash2 } from 'lucide-vue-next';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import {
  setListHandleDragImage,
  useListHandleReorder,
} from '@/web/web/composables/useListHandleReorder';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { createTranslate } from '@/web/web/i18n';
import { useI18nStore } from '@/web/web/stores/i18nStore';
import ChoyPropertiesDefinitionItemDialog from './ChoyPropertiesDefinitionItemDialog.vue';
import {
  cloneDraftItem,
  definitionItemsToDrafts,
  draftsToDefinitionItems,
  emptyDraftItem,
  type DefinitionEditorDraftItem,
} from './propertiesDefinitionHelpers';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyPropertiesDefinitionField' });

defineOptions({ name: 'ChoyPropertiesDefinitionField' });

/** Definition lines table: O2M-like frame with handle drag + footer “Add a line…”. */
const DefinitionLinesTable = defineComponent({
  name: 'ChoyPropertiesDefinitionLinesTable',
  props: {
    drafts: { type: Array as () => DefinitionEditorDraftItem[], required: true },
    canMutate: { type: Boolean, required: true },
    emptyText: { type: String, required: true },
    nameHeader: { type: String, required: true },
    typeHeader: { type: String, required: true },
    labelHeader: { type: String, required: true },
    removeText: { type: String, required: true },
    reorderText: { type: String, default: 'Reorder' },
    /** Index column header (O2M-style "#"). */
    indexHeader: { type: String, default: '#' },
    addText: { type: String, default: '' },
  },
  emits: {
    open: (_index: number, _asReadonly: boolean) => true,
    remove: (_index: number) => true,
    reorder: (_rows: DefinitionEditorDraftItem[]) => true,
    add: () => true,
  },
  setup(props, { emit }) {
    const handleEnabled = computed(() => props.canMutate);
    const handleReorder = useListHandleReorder({
      rows: () => props.drafts as DefinitionEditorDraftItem[],
      enabled: handleEnabled,
      // Drafts are not Sequence-backed; discard renumber mutations on throwaways.
      getRecord: () => ({}),
      onReorder: rows => emit('reorder', rows as DefinitionEditorDraftItem[]),
    });

    return () => {
      const header = h('thead', [
        h('tr', { class: 'border-b border-border bg-muted/40 text-left text-muted-foreground' }, [
          props.canMutate ? h('th', { class: 'w-9 px-1 py-1.5 font-medium' }) : null,
          h(
            'th',
            {
              class: 'w-11 whitespace-nowrap px-2 py-1.5 text-right font-medium',
              'data-testid': 'choy-properties-definition-index-header',
            },
            props.indexHeader,
          ),
          h('th', { class: 'whitespace-nowrap px-3 py-1.5 font-medium' }, props.nameHeader),
          h('th', { class: 'whitespace-nowrap px-3 py-1.5 font-medium' }, props.typeHeader),
          h('th', { class: 'min-w-[10rem] px-3 py-1.5 font-medium' }, props.labelHeader),
          // Actions (Edit/Remove) only in edit mode; display mode has no View column.
          props.canMutate ? h('th', { class: 'whitespace-nowrap px-3 py-1.5 font-medium' }) : null,
        ]),
      ]);

      const colCount = props.canMutate ? 6 : 4;
      const dragging = handleReorder.draggingIndex.value;
      const over = handleReorder.overIndex.value;

      const bodyRows =
        props.drafts.length === 0
          ? [
              h('tr', { 'data-testid': 'choy-properties-definition-empty' }, [
                h(
                  'td',
                  {
                    class: 'px-3 py-3 text-sm text-muted-foreground',
                    colspan: colCount,
                  },
                  props.emptyText,
                ),
              ]),
            ]
          : props.drafts.map((item, index) =>
              h(
                'tr',
                {
                  key: `${item.name || 'row'}-${index}`,
                  class: [
                    'cursor-pointer border-b border-border last:border-b-0 hover:bg-accent/40',
                    dragging === index ? 'opacity-60' : '',
                    dragging != null && over === index && over !== dragging
                      ? 'bg-accent/40 ring-1 ring-inset ring-primary/30'
                      : '',
                  ],
                  'data-index': index,
                  'data-testid': 'choy-properties-definition-row',
                  role: 'button',
                  tabindex: 0,
                  onClick: () => emit('open', index, !props.canMutate),
                  onKeydown: (e: KeyboardEvent) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      e.preventDefault();
                      emit('open', index, !props.canMutate);
                    }
                  },
                  onDragover: props.canMutate
                    ? (e: DragEvent) => {
                        e.preventDefault();
                        handleReorder.onDragOver(index, e);
                      }
                    : undefined,
                  onDrop: props.canMutate
                    ? (e: DragEvent) => {
                        e.preventDefault();
                        void handleReorder.onDrop(index, e);
                      }
                    : undefined,
                },
                [
                  props.canMutate
                    ? h(
                        'td',
                        {
                          class: 'w-9 px-1 py-1.5 align-middle',
                          onClick: (e: Event) => e.stopPropagation(),
                        },
                        [
                          h(
                            'span',
                            {
                              class:
                                'inline-flex size-7 cursor-grab items-center justify-center text-muted-foreground select-none active:cursor-grabbing',
                              draggable: 'true',
                              title: props.reorderText,
                              'aria-label': props.reorderText,
                              'data-testid': 'choy-properties-definition-handle',
                              onDragstart: (e: DragEvent) => {
                                e.stopPropagation();
                                handleReorder.onDragStart(index, e);
                                setListHandleDragImage(e);
                              },
                              onDragend: (e: DragEvent) => {
                                e.stopPropagation();
                                handleReorder.onDragEnd();
                              },
                              onClick: (e: Event) => e.stopPropagation(),
                            },
                            [h(GripVertical, { class: 'size-3.5', 'aria-hidden': 'true' })],
                          ),
                        ],
                      )
                    : null,
                  h(
                    'td',
                    {
                      class: 'w-11 whitespace-nowrap px-2 py-1.5 text-right tabular-nums text-muted-foreground',
                      'data-testid': 'choy-properties-definition-index',
                    },
                    String(index + 1),
                  ),
                  h(
                    'td',
                    {
                      class: 'max-w-[14rem] truncate px-3 py-1.5 font-mono text-xs',
                      title: item.name || undefined,
                    },
                    item.name || '—',
                  ),
                  h('td', { class: 'whitespace-nowrap px-3 py-1.5' }, item.type || '—'),
                  h(
                    'td',
                    {
                      class: 'max-w-[16rem] truncate px-3 py-1.5',
                      title: item.string || undefined,
                    },
                    item.string || '—',
                  ),
                  props.canMutate
                    ? h(
                        'td',
                        {
                          class: 'w-11 whitespace-nowrap px-1 py-1.5 text-center',
                          onClick: (e: Event) => e.stopPropagation(),
                          // Stop keydown so Enter/Space on Remove does not bubble to the
                          // row role=button handler (which would open the dialog instead).
                          onKeydown: (e: KeyboardEvent) => e.stopPropagation(),
                        },
                        [
                          h(
                            ChoyButton,
                            {
                              type: 'button',
                              size: 'icon',
                              variant: 'ghost',
                              class: 'size-7 text-muted-foreground hover:text-destructive',
                              'aria-label': props.removeText,
                              title: props.removeText,
                              'data-testid': 'choy-properties-definition-remove',
                              onClick: () => emit('remove', index),
                            },
                            () => h(Trash2, { class: 'size-3.5', 'aria-hidden': 'true' }),
                          ),
                        ],
                      )
                    : null,
                ],
              ),
            );

      const footer =
        props.canMutate && props.addText
          ? h(
              'div',
              {
                class: 'flex items-center border-t border-border px-3 py-2',
                'data-testid': 'choy-properties-definition-add-footer',
              },
              [
                h(
                  ChoyButton,
                  {
                    type: 'button',
                    size: 'sm',
                    variant: 'link',
                    class: 'h-auto px-0',
                    'data-testid': 'choy-properties-definition-add',
                    onClick: () => emit('add'),
                  },
                  () => props.addText,
                ),
              ],
            )
          : null;

      return h(
        'div',
        {
          class: 'w-full min-w-0 overflow-hidden rounded-md border border-border',
          'data-testid': 'choy-properties-definition-table',
        },
        [
          h('div', { class: 'w-full overflow-x-auto' }, [
            h('table', { class: 'w-full border-collapse text-sm' }, [
              header,
              h('tbody', bodyRows),
            ]),
          ]),
          footer,
        ],
      );
    };
  },
});

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;

    label?: string;
    rules?: RuleItem[];

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    formItemProps?: Partial<FormItemProps>;
    orientation?: 'horizontal' | 'vertical';
    /** Hide label row when Section/Tab already titles the block. Prefer over `label=""`. */
    hideLabel?: boolean;
    vColumnProps?: {
      width?: number | string;
      minWidth?: number;
      align?: 'left' | 'center' | 'right';
      fixed?: 'left' | 'right';
      sortable?: boolean;
    };
    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;
  }>(),
  {
    rules: () => [],
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    formItemProps: () => ({}),
    hideLabel: false,
    vColumnProps: () => ({}),
    renderMode: 'auto',
    showInlineError: false,
  },
);

const binding = (props.binding ??
  useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

const isTableLike = computed(() => {
  const mode = props.renderMode;
  if (mode === 'table' || mode === 'inline') return true;
  if (mode === 'form') return false;
  return binding.env?.isForm === false;
});

const canMutate = computed(() => Boolean(binding.env?.isEditMode));

function editorLang(): string {
  if (!getActivePinia()) return TRANSLATED_BASE_LANG;
  try {
    const lang = String(useI18nStore().terminologyLang || '').trim();
    return lang || TRANSLATED_BASE_LANG;
  } catch {
    return TRANSLATED_BASE_LANG;
  }
}

const toView = (raw: any): PropertyItemDefinition[] => (Array.isArray(raw) ? (raw as PropertyItemDefinition[]) : []);
const fromView = (view: PropertyItemDefinition[]) => view as unknown as V;

const drafts = ref<DefinitionEditorDraftItem[]>([]);
const previousItems = ref<PropertyItemDefinition[]>([]);
const writeError = ref('');
const dialogOpen = ref(false);
const dialogReadonly = ref(false);
const dialogIndex = ref<number | null>(null);
/** True when the open dialog is creating a new row (not editing by index). */
const dialogIsNew = ref(false);
/** Name of the row opened for edit; detects reorder while the dialog is open. */
const dialogOriginalName = ref<string | null>(null);
const dialogItem = ref<DefinitionEditorDraftItem | null>(null);

/** Suppress draft re-sync while we write back from local drafts. */
let syncingFromWrite = false;

function summaryText(raw: unknown): string {
  const items = toView(raw);
  if (!items.length) return _t('No properties');
  const names = items
    .map(it => String((it as any)?.name || '').trim())
    .filter(Boolean);
  if (!names.length) return _t('%s properties', String(items.length));
  if (names.length <= 3) return names.join(', ');
  return _t('%s properties (%s, …)', String(names.length), names.slice(0, 2).join(', '));
}

function syncDraftsFromRaw(raw: unknown) {
  const items = toView(raw);
  // External field updates replace drafts; an open edit dialog's dialogIndex may
  // then target a different row. Close it (new-item dialogs stay open).
  const editWasInvalidated = dialogOpen.value && !dialogIsNew.value && !dialogReadonly.value;
  if (editWasInvalidated) {
    dialogOpen.value = false;
    dialogIsNew.value = false;
    dialogIndex.value = null;
    dialogOriginalName.value = null;
    dialogItem.value = null;
  }
  previousItems.value = items;
  drafts.value = definitionItemsToDrafts(items, editorLang());
  writeError.value = editWasInvalidated
    ? _t('This row was changed while editing; please reopen it.')
    : '';
}

function commitDrafts(next: DefinitionEditorDraftItem[]) {
  writeError.value = '';
  // Clone so a failed draftsToDefinitionItems leave caller's drafts unmutated.
  const working = next.map(d => cloneDraftItem(d));
  let definition: PropertyItemDefinition[];
  try {
    definition = draftsToDefinitionItems(working, editorLang(), previousItems.value);
  } catch (e) {
    writeError.value = e instanceof Error ? e.message : String(e);
    throw e;
  }
  drafts.value = working;
  previousItems.value = definition;
  const fieldRef = binding.fieldRef() as WritableComputedRef<any> | { value: any };
  // Hold until the next tick so the deep fieldRef watcher does not re-sync
  // (and clear writeError) before Vue flushes the write.
  syncingFromWrite = true;
  try {
    fieldRef.value = definition;
  } finally {
    void nextTick(() => {
      syncingFromWrite = false;
    });
  }
}

function onAdd() {
  if (!canMutate.value) return;
  writeError.value = '';
  dialogReadonly.value = false;
  dialogIsNew.value = true;
  dialogIndex.value = null;
  dialogOriginalName.value = null;
  dialogItem.value = emptyDraftItem();
  dialogOpen.value = true;
}

function onRemove(index: number) {
  if (!canMutate.value) return;
  try {
    commitDrafts(drafts.value.filter((_, i) => i !== index));
  } catch {
    // writeError already set
  }
}

function onReorder(rows: DefinitionEditorDraftItem[]) {
  if (!canMutate.value) return;
  try {
    commitDrafts(rows);
  } catch {
    // writeError already set
  }
}

function openRow(index: number, asReadonly: boolean) {
  const item = drafts.value[index];
  if (!item) return;
  writeError.value = '';
  dialogReadonly.value = asReadonly || !canMutate.value;
  dialogIsNew.value = false;
  dialogIndex.value = index;
  dialogOriginalName.value = String(item.name || '');
  dialogItem.value = item;
  dialogOpen.value = true;
}

/** Promote a readonly preview to edit when the parent form can mutate. */
function onDialogEdit() {
  if (!canMutate.value || !dialogOpen.value) return;
  writeError.value = '';
  dialogReadonly.value = false;
}

function onDialogSaved(item: DefinitionEditorDraftItem) {
  if (dialogReadonly.value) return;
  const next = [...drafts.value];
  if (dialogIsNew.value) {
    next.push(item);
  } else {
    const index = dialogIndex.value;
    // Stale index (row removed/reordered while dialog open): keep dialog open.
    if (index == null || index < 0 || index >= next.length) {
      writeError.value = _t('This row was changed while editing; please reopen it.');
      return;
    }
    if (dialogOriginalName.value != null && next[index]?.name !== dialogOriginalName.value) {
      writeError.value = _t('This row was changed while editing; please reopen it.');
      return;
    }
    next[index] = item;
  }
  try {
    commitDrafts(next);
  } catch {
    // Keep dialog open so the user can fix invalid input without retyping.
    return;
  }
  dialogOpen.value = false;
  dialogIsNew.value = false;
  dialogIndex.value = null;
  dialogOriginalName.value = null;
  dialogItem.value = null;
}

watch(
  () => [binding.fieldRef().value, editorLang(), binding.env?.isEditMode] as const,
  () => {
    if (syncingFromWrite) return;
    syncDraftsFromRaw(binding.fieldRef().value);
  },
  { immediate: true, deep: true },
);

defineExpose({ drafts, previousItems, commitDrafts });
</script>
