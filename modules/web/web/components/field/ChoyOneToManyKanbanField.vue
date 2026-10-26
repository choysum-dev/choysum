<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :orientation="orientation"
    :hide-label="hideLabel"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
    :label-for-control="false"
  >
    <template #edit="{ inputId, labelId, describedBy, invalid }">
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-o2m-kanban-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope v-else :view-mode="binding.env.viewMode" :kind="'Kanban'" :field-prefix="String(prop)">
        <div
          class="flex w-full min-w-0 flex-col gap-2"
          :id="inputId || undefined"
          tabindex="-1"
          :aria-labelledby="labelId || undefined"
          :aria-describedby="describedBy || undefined"
          :aria-invalid="invalid ? 'true' : undefined"
        >
          <div v-if="showToolbar" class="flex items-center justify-start">
            <slot
              name="toolbar"
              :items="asKanbanRows(getItems())"
              :add="handleAddItem"
              :editable="editable"
            >
              <ChoyButton
                v-if="editable && showToolbarAdd"
                type="button"
                size="sm"
                variant="link"
                class="h-auto px-0"
                @click="handleAddItem"
              >{{ effectiveAddButtonText }}</ChoyButton>
            </slot>
          </div>

          <ChoyO2MKanbanVirtualBoard
            :items="kanbanItems"
            :min-card-width="minCardWidth"
            :gap="gap"
            :max-height="maxHeight"
            :virtualize="boardVirtualize"
            :item-key="kanbanItemKey"
          >
            <template #card="{ item, index }">
              <div
                class="choy-o2m-kanban-card group/card relative cursor-pointer rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/40"
                :class="{ 'opacity-60': sortableEnabled && kanbanReorder.draggingIndex.value === index }"
                data-testid="choy-o2m-kanban-card"
                role="button"
                tabindex="0"
                :aria-label="resolveCardTitle(item)"
                @click="handleCardClick(index, false)"
                @keydown.enter.prevent="handleCardClick(index, false)"
                @dragover="sortableEnabled ? kanbanReorder.onDragOver(index, $event) : undefined"
                @drop="sortableEnabled ? kanbanReorder.onDrop(index, $event) : undefined"
              >
                <!-- Host chrome outside #card: drag handle + hover delete. -->
                <button
                  v-if="sortableEnabled"
                  type="button"
                  class="absolute left-1 top-1.5 z-10 inline-flex size-7 cursor-grab items-center justify-center rounded-md text-muted-foreground opacity-0 transition-opacity hover:bg-muted hover:text-foreground group-hover/card:opacity-100 group-focus-within/card:opacity-100 active:cursor-grabbing"
                  :aria-label="_t('Reorder')"
                  data-testid="choy-o2m-kanban-handle"
                  draggable="true"
                  @click.stop
                  @dragstart="onKanbanDragStart(index, $event)"
                  @dragend="kanbanReorder.onDragEnd()"
                >
                  <GripVertical class="size-3.5" aria-hidden="true" />
                </button>
                <ChoyButton
                  v-if="removable"
                  type="button"
                  size="icon"
                  variant="ghost"
                  class="absolute right-1.5 top-1.5 z-10 size-7 text-muted-foreground opacity-0 transition-opacity hover:bg-destructive/10 hover:text-destructive group-hover/card:opacity-100 group-focus-within/card:opacity-100"
                  :aria-label="_t('Delete')"
                  data-testid="choy-o2m-kanban-delete"
                  @click.stop="handleRemoveItem(index)"
                >
                  <Trash2 class="size-3.5" aria-hidden="true" />
                </ChoyButton>
                <div
                  :class="[
                    'min-w-0',
                    removable || sortableEnabled ? 'pr-8' : '',
                    sortableEnabled ? 'pl-7' : '',
                  ]"
                >
                  <slot
                    name="card"
                    :item="asKanbanRow(item)"
                    :index="index"
                    :open="() => handleCardClick(index, false)"
                    :edit="() => handleEditItem(index)"
                    :remove="() => handleRemoveItem(index)"
                    :editable="editable"
                    :removable="removable"
                  >
                    <div class="text-sm font-medium leading-snug text-foreground">{{ resolveCardTitle(item) }}</div>
                    <div
                      v-if="resolveCardSubtitle(item)"
                      class="mt-1 truncate text-xs text-muted-foreground"
                    >
                      {{ resolveCardSubtitle(item) }}
                    </div>
                  </slot>
                </div>
              </div>
            </template>

            <template v-if="!editable" #empty>
              <div
                class="col-span-full flex min-h-[72px] items-center justify-center rounded-md border border-dashed border-border text-sm text-muted-foreground"
              >
                <slot name="empty">{{ effectiveEmptyText }}</slot>
              </div>
            </template>

            <template v-if="editable" #trailing>
              <div
                class="flex min-h-[72px] cursor-pointer select-none items-center justify-center rounded-md border border-dashed border-border bg-muted/30 text-sm text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
                data-testid="choy-o2m-kanban-add"
                role="button"
                tabindex="0"
                @click="handleAddItem"
                @keydown.enter.prevent="handleAddItem"
              >
                <span>{{ effectiveAddButtonText }}</span>
              </div>
            </template>
          </ChoyO2MKanbanVirtualBoard>
        </div>
      </ChoyViewScope>
    </template>

    <template #display="{ inputId, labelId }">
      <span
        v-if="showCellSummary"
        class="truncate text-sm text-foreground"
        data-testid="choy-o2m-kanban-summary"
      >{{ cellSummaryText }}</span>
      <ChoyViewScope v-else view-mode="display" :kind="'Kanban'" :field-prefix="String(prop)">
        <div
          class="flex w-full min-w-0 flex-col gap-2"
          :id="inputId || undefined"
          :aria-labelledby="labelId || undefined"
        >
          <ChoyO2MKanbanVirtualBoard
            :items="kanbanItems"
            :min-card-width="minCardWidth"
            :gap="gap"
            :max-height="maxHeight"
            :virtualize="boardVirtualize"
            :item-key="kanbanItemKey"
          >
            <template #card="{ item, index }">
              <div
                class="choy-o2m-kanban-card cursor-default rounded-md border border-border bg-card p-3 shadow-xs outline-none transition-colors hover:bg-accent/30"
                data-testid="choy-o2m-kanban-card"
                role="button"
                tabindex="0"
                :aria-label="resolveCardTitle(item)"
                @click="handleCardClick(index, true)"
                @keydown.enter.prevent="handleCardClick(index, true)"
              >
                <slot
                  name="card"
                  :item="asKanbanRow(item)"
                  :index="index"
                  :open="() => handleCardClick(index, true)"
                  :edit="() => {}"
                  :remove="noopRemove"
                  :editable="false"
                  :removable="false"
                >
                  <div class="text-sm font-medium leading-snug text-foreground">{{ resolveCardTitle(item) }}</div>
                  <div
                    v-if="resolveCardSubtitle(item)"
                    class="mt-1 truncate text-xs text-muted-foreground"
                  >
                    {{ resolveCardSubtitle(item) }}
                  </div>
                </slot>
              </div>
            </template>

            <template v-if="$slots.empty" #empty>
              <div
                class="col-span-full flex min-h-[72px] items-center justify-center rounded-md border border-dashed border-border text-sm text-muted-foreground"
              >
                <slot name="empty" />
              </div>
            </template>
          </ChoyO2MKanbanVirtualBoard>
        </div>
      </ChoyViewScope>
    </template>
  </ChoyFieldBase>

  <ChoyDialog v-model:open="dialogVisible">
    <ChoyDialogContent
      class="choy-relation-picker-dialog flex max-h-[min(85vh,40rem)] w-[min(90vw,48rem)] max-w-[min(90vw,48rem)] flex-col gap-3 overflow-hidden"
      :style="{ width: typeof dialogWidth === 'number' ? dialogWidth + 'px' : dialogWidth }"
    >
      <ChoyDialogTitle class="shrink-0">{{ dialogTitleText }}</ChoyDialogTitle>
      <div class="min-h-0 flex-1 overflow-auto">
        <component
          v-if="formView"
          :is="formView"
          ref="dialogFormRef"
          :key="dialogFormKey"
          :store="dialogStore"
          :view-mode="dialogFormMode"
          :resolve-record-id-from-route="false"
          :initial-values="dialogDraft"
          :show-header="false"
          :show-actions="false"
          :show-messages="false"
          :submit-handler="dialogFormSubmitHandler"
          v-bind="formViewProps"
        />
        <div
          v-else
          class="rounded-md border border-dashed border-border p-4 text-sm text-muted-foreground"
        >{{ _t('Provide a child record editor via the formView prop.') }}</div>
      </div>
      <div class="flex shrink-0 items-center justify-between gap-2">
        <div class="min-w-0">
          <ChoyButton
            v-if="dialogMode === 'edit' && removable"
            type="button"
            variant="ghost"
            class="text-destructive hover:bg-destructive/10 hover:text-destructive"
            data-testid="choy-o2m-kanban-dialog-delete"
            @click="handleDialogRemove"
          >{{ _t('Delete') }}</ChoyButton>
        </div>
        <div class="flex shrink-0 justify-end gap-2">
          <ChoyButton type="button" variant="outline" @click="handleDialogCancel">{{ _t('Cancel') }}</ChoyButton>
          <ChoyButton v-if="dialogMode !== 'display'" type="button" @click="handleDialogSubmit">{{ _t('Save') }}</ChoyButton>
        </div>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, watch, onMounted, provide, useSlots, nextTick, type Component } from 'vue';
import { GripVertical, Trash2 } from 'lucide-vue-next';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { confirmChoyAction } from '../../composables/confirmChoyAction';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { deepClonePreserve } from '@/core/utils/clone';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import ChoyO2MKanbanVirtualBoard from './ChoyO2MKanbanVirtualBoard.vue';
import type { O2MKanbanVirtualizeMode } from './o2mKanbanVirtualHelpers';
import ChoyViewScope from '@/web/web/components/view/ChoyViewScope.vue';
import {
  type FormChildSubmitApi,
  type FormChildSubmitApiRegistration,
  type FormSubmitOutcome,
  type FormSubmitHandler,
  type FormSubmitHandlerContext
} from '@/web/web/components/view/formViewTypes';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';
import { hasHandleField } from '@/web/web/composables/listRowEdit';
import { useListHandleReorder } from '@/web/web/composables/useListHandleReorder';
import { formatRelationCountSummary } from './relationCountSummary';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyOneToManyKanbanField' });

defineOptions({ name: 'ChoyOneToManyKanbanField', inheritAttrs: false });

type IsAny<TA> = 0 extends 1 & TA ? true : false;
type DialogMode = 'create' | 'edit' | 'display';
/** One O2M row in the kanban board / `#card` slot (element of relation array `V`). */
type KanbanRow = [V] extends [never]
  ? Record<string, unknown>
  : NonNullable<V> extends readonly (infer U)[] | (infer U)[]
    ? U
    : IsAny<V> extends true
      ? any
      : Record<string, unknown>;
const FORM_CHILD_SUBMIT_API_REGISTER_KEY = 'form-child-submit-api-register';
const FORM_EMBEDDED_CONTEXT_KEY = 'form-embedded-context';

// Untyped slots collapse `#card` item to `{}` in consumers (e.g. SandboxRecordFormView).
// Must match `<slot name="card|toolbar|empty" …>` bindings below.
defineSlots<{
  card(props: {
    item: KanbanRow;
    index: number;
    open: () => void;
    edit: () => void;
    /** Resolves true when the row was removed; false when blocked or cancelled. */
    remove: () => void | Promise<boolean>;
    editable: boolean;
    removable: boolean;
  }): unknown;
  toolbar(props: {
    items: KanbanRow[];
    add: () => void;
    editable: boolean;
  }): unknown;
  empty(): unknown;
}>();

/** Board items are loosely typed; narrow for slot consumers / defineSlots checks. */
function asKanbanRow(item: unknown): KanbanRow {
  return item as KanbanRow;
}

function asKanbanRows(items: unknown): KanbanRow[] {
  return (Array.isArray(items) ? items : []) as KanbanRow[];
}

/** Display-mode stub: removal is disabled; keeps `#card` remove typing honest. */
function noopRemove(): Promise<boolean> {
  return Promise.resolve(false);
}

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;
    /** Subtable default: label above the kanban block (full width). */
    orientation?: 'horizontal' | 'vertical';
    /** Hide label row when Section/Tab already titles the block. Prefer over `label=""`. */
    hideLabel?: boolean;
    defaultRecord?: Record<string, any> | (() => Record<string, any>);

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;

    cardTitleField?: string;
    cardSubtitleField?: string;
    /**
     * Extra child field names (or `Prop.Child` paths) to include in Form returnFields.
     * Needed when a custom `#card` slot reads columns that are not title/subtitle — Tabs
     * unmount inactive panes, so the sibling O2M table fields are not registered on save.
     */
    fields?: string[];
    addButtonText?: string;
    showToolbarAdd?: boolean;
    formView?: Component;
    formViewProps?: Record<string, unknown>;
    emptyText?: string;

    minCardWidth?: number;
    gap?: number;
    maxHeight?: number;
    /** Board virtualize mode; default `auto` (≥40 cards). */
    boardVirtualize?: O2MKanbanVirtualizeMode;

    editable?: boolean;
    removable?: boolean;
    previewInDisplay?: boolean;
    confirmOnRemove?: boolean;
    /** Sequence field for card drag reorder; default Sequence. */
    handleField?: string;
    /** Show drag handle when relation metadata includes handleField. */
    showHandle?: boolean;

    dialogWidth?: string | number;
    createDialogTitle?: string;
    editDialogTitle?: string;
    displayDialogTitle?: string;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    orientation: 'vertical',
    hideLabel: false,
    defaultRecord: () => ({}),
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    renderMode: 'auto',
    showInlineError: false,

    cardTitleField: 'Name',
    cardSubtitleField: 'Email',
    fields: () => [],
    addButtonText: '',
    showToolbarAdd: false,
    formView: undefined,
    formViewProps: () => ({}),
    emptyText: '',

    minCardWidth: 220,
    gap: 10,
    maxHeight: 480,
    boardVirtualize: 'auto',

    editable: true,
    removable: true,
    previewInDisplay: true,
    confirmOnRemove: true,
    handleField: 'Sequence',
    showHandle: true,

    dialogWidth: '760px',
    createDialogTitle: '',
    editDialogTitle: '',
    displayDialogTitle: '',
  }
);

const emit = defineEmits<{
  (e: 'add-click', payload: { defaultItem: Record<string, any> }): void;
  (e: 'card-click', payload: { index: number; item: any; mode: DialogMode }): void;
  (e: 'edit-request', payload: { index: number; item: any }): void;
  (e: 'remove-request', payload: { index: number; item: any }): void;
  (e: 'save-request', payload: { mode: 'create' | 'edit'; index: number; item: any }): void;
}>();

const effectiveAddButtonText = computed(() => props.addButtonText || _t('Add a line…'));
const effectiveEmptyText = computed(() => props.emptyText || _t('No data'));
const effectiveCreateDialogTitle = computed(() => props.createDialogTitle || _t('New record'));
const effectiveEditDialogTitle = computed(() => props.editDialogTitle || _t('Edit record'));
const effectiveDisplayDialogTitle = computed(() => props.displayDialogTitle || _t('View record'));

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;

/** Register nested O2M columns so Create/Update returnFields keep card data after save. */
function resolveKanbanReturnFieldPaths(): string[] {
  const prefix = String(binding.prop || props.prop || '').trim();
  if (!prefix) return [];
  const out = new Set<string>([`${prefix}.DisplayName`]);
  const title = String(props.cardTitleField || '').trim();
  const subtitle = String(props.cardSubtitleField || '').trim();
  if (title) out.add(`${prefix}.${title}`);
  if (subtitle) out.add(`${prefix}.${subtitle}`);
  for (const raw of props.fields || []) {
    const name = String(raw || '').trim();
    if (!name) continue;
    out.add(name.includes('.') ? name : `${prefix}.${name}`);
  }
  return [...out];
}
binding.registerFields(resolveKanbanReturnFieldPaths());

const { getItems, insertItem, removeItemAt } = binding.asMutableArray<any>();
const dialogStore = computed<WebModelStore<any>>(() => (binding.relationStore as WebModelStore<any>) || (props.store as WebModelStore<any>));
const slots = useSlots();

const showHandleColumn = computed(
  () => props.showHandle !== false && hasHandleField(binding.relationStore, props.handleField)
);
const sortableEnabled = computed(() => showHandleColumn.value && props.editable && binding.env.isEditMode);
const kanbanReorder = useListHandleReorder({
  rows: () => getItems(),
  enabled: sortableEnabled,
  handleField: props.handleField,
  onReorder: rows => {
    const ref = binding.fieldRef?.() as { value: any } | undefined;
    if (!ref) return;
    ref.value = rows.slice();
  },
});

// Keep Sequence in returnFields when drag reorder is available.
if (props.handleField) {
  binding.registerFields(`${String(binding.prop)}.${props.handleField}`);
}

const itemCount = computed(() => getItems()?.length ?? 0);
/** Explicit length dep so the board refreshes when the collection is replaced. */
const kanbanItems = computed(() => {
  const items = getItems() || [];
  void items.length;
  return items;
});
// List/Kanban cells: count summary instead of embedding a full nested board.
const showCellSummary = computed(() => binding.env?.isForm === false);
const cellSummaryText = computed(() => formatRelationCountSummary(itemCount.value, _t));

const showToolbar = computed(() => Boolean(slots.toolbar) || (props.editable && props.showToolbarAdd));

function kanbanItemKey(item: unknown, index: number): string {
  return String(readRowKeySeed(item) ?? index);
}

/** Snapshot drag image from the card node so virtual recycle does not blank it. */
function onKanbanDragStart(index: number, event: DragEvent) {
  kanbanReorder.onDragStart(index, event);
  const handle = event.currentTarget;
  const card =
    handle instanceof HTMLElement ? handle.closest('[data-testid=choy-o2m-kanban-card]') : null;
  if (card instanceof HTMLElement && event.dataTransfer) {
    const rect = card.getBoundingClientRect();
    const offsetX = Math.max(0, Math.min(rect.width, event.clientX - rect.left));
    const offsetY = Math.max(0, Math.min(rect.height, event.clientY - rect.top));
    event.dataTransfer.setDragImage(card, offsetX, offsetY);
  }
}

const dialogVisible = ref(false);
const dialogMode = ref<DialogMode>('create');
const dialogIndex = ref(-1);
const dialogDraft = ref<Record<string, any>>({});
const dialogFormKey = ref(0);
const dialogFormRef = ref<{ submit?: () => Promise<unknown>; getFormData?: () => unknown } | null>(null);
const dialogRegisteredFormApis = new Map<string, FormChildSubmitApi>();
const activeDialogFormToken = ref<string | null>(null);
const dialogFormMode = computed(() => (dialogMode.value === 'display' ? 'display' : 'create'));
const dialogTitleText = computed(() => {
  if (dialogMode.value === 'create') return effectiveCreateDialogTitle.value;
  if (dialogMode.value === 'edit') return effectiveEditDialogTitle.value;
  return effectiveDisplayDialogTitle.value;
});

provide(FORM_CHILD_SUBMIT_API_REGISTER_KEY, (registration: FormChildSubmitApiRegistration) => {
  const token = String(registration?.token || '').trim();
  if (!token) return;

  if (registration.api) {
    dialogRegisteredFormApis.set(token, registration.api);
    activeDialogFormToken.value = token;
    return;
  }

  const isActive = activeDialogFormToken.value === token;
  dialogRegisteredFormApis.delete(token);
  if (isActive) {
    const lastToken = Array.from(dialogRegisteredFormApis.keys()).pop() || null;
    activeDialogFormToken.value = lastToken;
  }
});
provide<boolean>(FORM_EMBEDDED_CONTEXT_KEY, true);

function getRegisteredDialogFormApi(): FormChildSubmitApi | null {
  const token = activeDialogFormToken.value;
  if (!token) return null;
  return dialogRegisteredFormApis.get(token) || null;
}

function readRowKeySeed(row: unknown): string | number | undefined {
  if (!row || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  return r.__rowKey ?? r.Id;
}

function defineHiddenRowKey(obj: any, key: string, val?: any) {
  if (!obj || typeof obj !== 'object') return;
  try {
    const hasOwn = Object.prototype.hasOwnProperty.call(obj, key);
    const enumerable = hasOwn ? Object.prototype.propertyIsEnumerable.call(obj, key) : false;
    if (!hasOwn) {
      Object.defineProperty(obj, key, {
        value: String(val ?? Math.random().toString(36).slice(2)),
        enumerable: false,
        configurable: false,
        writable: true,
      });
      return;
    }
    if (enumerable) {
      const v = val ?? obj[key];
      delete obj[key];
      Object.defineProperty(obj, key, {
        value: String(v ?? Math.random().toString(36).slice(2)),
        enumerable: false,
        configurable: false,
        writable: true,
      });
    }
  } catch {}
}

function hydrateRowKeys() {
  const arr = getItems() || [];
  for (const row of arr) {
    if (!row) continue;
    const seed = readRowKeySeed(row);
    defineHiddenRowKey(row, '__rowKey', seed);
  }
}

function ensureArrayInitialized() {
  const refVal = binding.fieldRef() as any;
  if (!Array.isArray(refVal.value)) {
    refVal.value = [];
  }
}

function makeDefaultItem(): Record<string, any> {
  const rec = typeof props.defaultRecord === 'function' ? (props.defaultRecord as any)() : props.defaultRecord;
  const row = { ...(rec || {}) };
  const seed = readRowKeySeed(row) ?? Math.random().toString(36).slice(2);
  defineHiddenRowKey(row, '__rowKey', seed);
  return row;
}

function resolveCardTitle(item: any): string {
  const field = String(props.cardTitleField || '').trim();
  const value = field ? item?.[field] : undefined;
  if (value != null && String(value).trim()) return String(value).trim();
  return String(item?.DisplayName || item?.Id || _t('Untitled'));
}

function resolveCardSubtitle(item: any): string {
  const field = String(props.cardSubtitleField || '').trim();
  const value = field ? item?.[field] : undefined;
  return value != null ? String(value) : '';
}

function openDialog(mode: DialogMode, index: number, item: any) {
  dialogMode.value = mode;
  dialogIndex.value = index;
  dialogDraft.value = deepClonePreserve((item || {}) as any);
  dialogRegisteredFormApis.clear();
  activeDialogFormToken.value = null;
  dialogFormKey.value += 1;
  dialogVisible.value = true;
}

function handleAddItem() {
  if (!props.editable) return;
  const row = makeDefaultItem();
  emit('add-click', { defaultItem: deepClonePreserve(row) });
  openDialog('create', -1, row);
}

function handleCardClick(index: number, displayTrigger: boolean) {
  const item = getItems()[index];
  if (!item) return;

  const mode: DialogMode = displayTrigger || !binding.env.isEditMode || !props.editable ? 'display' : 'edit';
  if (mode === 'display' && !props.previewInDisplay) return;

  emit('card-click', { index, item, mode });
  openDialog(mode, index, item);
}

function handleEditItem(index: number) {
  if (!props.editable) return;
  const item = getItems()[index];
  if (!item) return;
  emit('edit-request', { index, item });
  openDialog('edit', index, item);
}

async function handleRemoveItem(index: number): Promise<boolean> {
  if (!props.removable) return false;
  const item = getItems()[index];
  if (!item) return false;

  const doRemove = async () => {
    emit('remove-request', { index, item });
    removeItemAt(index);
  };

  if (!props.confirmOnRemove) {
    await doRemove();
    return true;
  }

  try {
    await confirmChoyAction(
      _t('Are you sure you want to delete this record? This action cannot be undone.'),
      _t('Confirm delete'),
      {
        confirmText: _t('Delete'),
        cancelText: _t('Cancel'),
        destructive: true,
      }
    );
    await doRemove();
    return true;
  } catch {
    return false;
  }
}

function handleDialogCancel() {
  dialogVisible.value = false;
}

async function handleDialogRemove() {
  if (dialogMode.value !== 'edit' || !props.removable) return;
  const index = dialogIndex.value;
  if (index < 0) return;
  const item = getItems()[index];
  if (!item) return;

  // Close the edit Dialog before confirm: Dialog + AlertDialog both stack at
  // z-50 with modal focus traps; opening confirm underneath freezes the page.
  const reopenDraft = deepClonePreserve((dialogDraft.value || item) as Record<string, any>);
  dialogVisible.value = false;
  await nextTick();

  const removed = await handleRemoveItem(index);
  if (!removed) {
    openDialog('edit', index, reopenDraft);
  }
}

const dialogFormSubmitHandler: FormSubmitHandler<any> = async (ctx: FormSubmitHandlerContext<any>) => {
  return {
    handled: true,
    record: deepClonePreserve((ctx.formData || {}) as any),
    skipSuccessMessage: true,
  };
};

async function handleDialogSubmit() {
  if (dialogMode.value === 'display') {
    dialogVisible.value = false;
    return;
  }

  const formRef = getRegisteredDialogFormApi() || dialogFormRef.value;
  if (!formRef?.submit) {
    const payload = (formRef?.getFormData?.() as Record<string, any>) || dialogDraft.value;
    handleDialogSave(payload);
    return;
  }

  const submitResult = await formRef.submit();
  if (submitResult && typeof submitResult === 'object' && 'ok' in submitResult) {
    const outcome = submitResult as FormSubmitOutcome<any>;
    if (!outcome.ok) return;
    const payload = (outcome.record || outcome.formData || dialogDraft.value || {}) as Record<string, any>;
    handleDialogSave(payload);
    return;
  }

  if (submitResult === false) return;
  const payload = (formRef.getFormData?.() as Record<string, any>) || dialogDraft.value;
  handleDialogSave(payload);
}

function handleDialogSave(payload?: Record<string, any>) {
  if (dialogMode.value === 'display') {
    dialogVisible.value = false;
    return;
  }

  const nextItem = deepClonePreserve(((payload || dialogDraft.value || {}) as any) || {});
  const existingRowKey = readRowKeySeed(nextItem) ?? readRowKeySeed(dialogDraft.value);
  defineHiddenRowKey(nextItem, '__rowKey', existingRowKey ?? Math.random().toString(36).slice(2));

  if (dialogMode.value === 'create') {
    ensureArrayInitialized();
    insertItem(nextItem);
    emit('save-request', {
      mode: 'create',
      index: getItems().length - 1,
      item: deepClonePreserve(nextItem),
    });
  } else if (dialogMode.value === 'edit' && dialogIndex.value >= 0) {
    const arr = [...(getItems() || [])];
    arr.splice(dialogIndex.value, 1, nextItem);
    (binding.fieldRef() as any).value = arr as any;
    emit('save-request', {
      mode: 'edit',
      index: dialogIndex.value,
      item: deepClonePreserve(nextItem),
    });
  }

  dialogVisible.value = false;
}

onMounted(hydrateRowKeys);
watch(
  () => getItems().length,
  () => hydrateRowKeys(),
  { immediate: true }
);
</script>

