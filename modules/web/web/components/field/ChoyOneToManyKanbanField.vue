<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FieldBase
    v-bind="$attrs"
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
    :required="required"
    :readonly="readonly"
    :visible="visible"
    :cellVisible="cellVisible"
    :renderMode="renderMode"
    :showInlineError="showInlineError"
  >
    <template #edit>
      <ChoyViewScope :view-mode="binding.env.viewMode" :container="'Kanban'" :field-prefix="String(prop)">
        <div class="flex w-full min-w-0 flex-col gap-2.5">
          <div v-if="showToolbar" class="flex items-center justify-start">
            <slot name="toolbar" :items="getItems()" :add="handleAddItem" :editable="editable">
              <ChoyButton v-if="editable && showToolbarAdd" size="sm" variant="link" @click="handleAddItem">{{ effectiveAddButtonText }}</ChoyButton>
            </slot>
          </div>

          <div class="grid w-full min-w-0 overflow-auto p-0.5 [grid-template-columns:repeat(auto-fill,minmax(var(--kanban-min-card-width),1fr))] gap-[var(--kanban-gap)]" :style="boardStyle">
            <template v-if="getItems().length > 0">
              <div
                v-for="(item, index) in getItems()"
                :key="String(readRowKeySeed(item) ?? index)"
                class="cursor-pointer rounded-lg border border-border bg-background p-3 transition-[border-color,box-shadow] duration-150 hover:border-primary-soft hover:shadow-sm"
                @click="handleCardClick(index, false)"
              >
                <slot
                  name="card"
                  :item="item"
                  :index="index"
                  :open="() => handleCardClick(index, false)"
                  :edit="() => handleEditItem(index)"
                  :remove="() => handleRemoveItem(index)"
                  :editable="editable"
                  :removable="removable"
                >
                  <div class="text-sm font-semibold leading-snug text-foreground">{{ resolveCardTitle(item) }}</div>
                  <div class="mt-1.5 break-words text-xs text-muted-foreground" v-if="resolveCardSubtitle(item)">{{ resolveCardSubtitle(item) }}</div>
                  <div class="mt-2.5 flex items-center gap-2" v-if="editable || removable" @click.stop>
                    <ChoyButton v-if="editable" size="sm" variant="link" @click="handleEditItem(index)">{{ _t('Edit') }}</ChoyButton>
                    <ChoyButton v-if="removable" size="sm" variant="destructive" @click="handleRemoveItem(index)">{{ _t('Delete') }}</ChoyButton>
                  </div>
                </slot>
              </div>
            </template>

            <div v-else class="col-span-full flex min-h-[90px] items-center justify-center rounded-lg border border-dashed border-border text-[13px] text-muted-foreground">
              <slot name="empty">{{ effectiveEmptyText }}</slot>
            </div>

            <div v-if="editable" class="flex min-h-24 cursor-pointer select-none items-center justify-center rounded-lg border border-dashed border-border bg-primary/5 text-primary hover:border-primary" @click="handleAddItem">
              <span>{{ effectiveAddButtonText }}</span>
            </div>
          </div>
        </div>
      </ChoyViewScope>
    </template>

    <template #display>
      <ChoyViewScope view-mode="display" :container="'Kanban'" :field-prefix="String(prop)">
        <div class="flex w-full min-w-0 flex-col gap-2.5">
          <div class="grid w-full min-w-0 overflow-auto p-0.5 [grid-template-columns:repeat(auto-fill,minmax(var(--kanban-min-card-width),1fr))] gap-[var(--kanban-gap)]" :style="boardStyle">
            <template v-if="getItems().length > 0">
              <div
                v-for="(item, index) in getItems()"
                :key="String(readRowKeySeed(item) ?? index)"
                class="cursor-pointer rounded-lg border border-border bg-background p-3 transition-[border-color,box-shadow] duration-150 hover:border-primary-soft hover:shadow-sm"
                @click="handleCardClick(index, true)"
              >
                <slot
                  name="card"
                  :item="item"
                  :index="index"
                  :open="() => handleCardClick(index, true)"
                  :edit="() => {}"
                  :remove="() => {}"
                  :editable="false"
                  :removable="false"
                >
                  <div class="text-sm font-semibold leading-snug text-foreground">{{ resolveCardTitle(item) }}</div>
                  <div class="mt-1.5 break-words text-xs text-muted-foreground" v-if="resolveCardSubtitle(item)">{{ resolveCardSubtitle(item) }}</div>
                </slot>
              </div>
            </template>

            <div v-else class="col-span-full flex min-h-[90px] items-center justify-center rounded-lg border border-dashed border-border text-[13px] text-muted-foreground">
              <slot name="empty">{{ effectiveEmptyText }}</slot>
            </div>
          </div>
        </div>
      </ChoyViewScope>
    </template>
  </FieldBase>

  <ChoyDialog v-model:open="dialogVisible">
    <ChoyDialogContent class="choy-relation-picker-dialog" :style="{ width: typeof dialogWidth === 'number' ? dialogWidth + 'px' : dialogWidth }">
      <ChoyDialogTitle>{{ dialogTitleText }}</ChoyDialogTitle>
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
    <div v-else class="rounded-lg border border-dashed border-border p-[18px] text-[13px] text-muted-foreground">{{ _t('Provide a child record editor via the formView prop.') }}</div>
      <div class="dialog-footer">
        <ChoyButton @click="handleDialogCancel">{{ _t('Cancel') }}</ChoyButton>
        <ChoyButton v-if="dialogMode !== 'display'" @click="handleDialogSubmit">{{ _t('Save') }}</ChoyButton>
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, watch, onMounted, provide, useSlots, type Component } from 'vue';
import { ChoyDialog, ChoyDialogContent, ChoyDialogTitle } from '@/web/web/components/layout/choyDialog';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import { confirmChoyAction, confirmChoyChoice } from '../../composables/confirmChoyAction';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { deepClonePreserve } from '@/core/utils/clone';
import FieldBase, { type FieldStateExpr, type FormItemProps } from './FieldBase.vue';
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

const { _t } = createTranslate('web', { scope: 'web/components/field/OneToManyKanbanField' });

defineOptions({ name: 'OneToManyKanbanField', inheritAttrs: false });

type IsAny<TA> = 0 extends 1 & TA ? true : false;
type DialogMode = 'create' | 'edit' | 'display';
const FORM_CHILD_SUBMIT_API_REGISTER_KEY = 'form-child-submit-api-register';
const FORM_EMBEDDED_CONTEXT_KEY = 'form-embedded-context';

const props = withDefaults(
  defineProps<{
    store?: WebModelStore<T>;
    prop?: P | (IsAny<T> extends true ? string : never);
    binding?: UseField<T, V>;
    label?: string;
    rules?: RuleItem[];
    formItemProps?: Partial<FormItemProps>;
    defaultRecord?: Record<string, any> | (() => Record<string, any>);

    required?: FieldStateExpr<T, V>;
    readonly?: FieldStateExpr<T, V>;
    visible?: FieldStateExpr<T, V>;
    cellVisible?: FieldStateExpr<T, V>;

    renderMode?: 'auto' | 'form' | 'table' | 'inline';
    showInlineError?: boolean;

    cardTitleField?: string;
    cardSubtitleField?: string;
    addButtonText?: string;
    showToolbarAdd?: boolean;
    formView?: Component;
    formViewProps?: Record<string, unknown>;
    emptyText?: string;

    minCardWidth?: number;
    gap?: number;
    maxHeight?: number;

    editable?: boolean;
    removable?: boolean;
    previewInDisplay?: boolean;
    confirmOnRemove?: boolean;

    dialogWidth?: string | number;
    createDialogTitle?: string;
    editDialogTitle?: string;
    displayDialogTitle?: string;
  }>(),
  {
    rules: () => [],
    formItemProps: () => ({}),
    defaultRecord: () => ({}),
    required: false,
    readonly: false,
    visible: true,
    cellVisible: true,
    renderMode: 'auto',
    showInlineError: false,

    cardTitleField: 'Name',
    cardSubtitleField: 'Email',
    addButtonText: '',
    showToolbarAdd: false,
    formView: undefined,
    formViewProps: () => ({}),
    emptyText: '',

    minCardWidth: 240,
    gap: 12,
    maxHeight: 480,

    editable: true,
    removable: true,
    previewInDisplay: true,
    confirmOnRemove: true,

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

const effectiveAddButtonText = computed(() => props.addButtonText || _t('New'));
const effectiveEmptyText = computed(() => props.emptyText || _t('No data'));
const effectiveCreateDialogTitle = computed(() => props.createDialogTitle || _t('New record'));
const effectiveEditDialogTitle = computed(() => props.editDialogTitle || _t('Edit record'));
const effectiveDisplayDialogTitle = computed(() => props.displayDialogTitle || _t('View record'));

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P })) as UseField<T, V>;
const { getItems, insertItem, removeItemAt } = binding.asMutableArray<any>();
const dialogStore = computed<WebModelStore<any>>(() => (binding.relationStore as WebModelStore<any>) || (props.store as WebModelStore<any>));
const slots = useSlots();

const boardStyle = computed(() => ({
  '--kanban-min-card-width': `${props.minCardWidth}px`,
  '--kanban-gap': `${props.gap}px`,
  maxHeight: `${props.maxHeight}px`,
}));
const showToolbar = computed(() => Boolean(slots.toolbar) || (props.editable && props.showToolbarAdd));

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

async function handleRemoveItem(index: number) {
  if (!props.removable) return;
  const item = getItems()[index];
  if (!item) return;

  const doRemove = async () => {
    emit('remove-request', { index, item });
    removeItemAt(index);
  };

  if (!props.confirmOnRemove) {
    await doRemove();
    return;
  }

  await confirmChoyAction(
    _t('Are you sure you want to delete this record? This action cannot be undone.'),
    _t('Confirm delete'),
    {
      confirmText: _t('Delete'),
      cancelText: _t('Cancel'),
      destructive: true,
    }
  )
    .then(doRemove)
    .catch(() => {});
}

function handleDialogCancel() {
  dialogVisible.value = false;
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

