<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
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
      <OViewScope :view-mode="binding.env.viewMode" :container="'Kanban'" :field-prefix="String(prop)">
        <div class="o-otm-kanban">
          <div v-if="showToolbar" class="o-otm-kanban__toolbar">
            <slot name="toolbar" :items="getItems()" :add="handleAddItem" :editable="editable">
              <el-button v-if="editable && showToolbarAdd" size="small" link type="primary" plain @click="handleAddItem">{{ effectiveAddButtonText }}</el-button>
            </slot>
          </div>

          <div class="o-otm-kanban__board" :style="boardStyle">
            <template v-if="getItems().length > 0">
              <div
                v-for="(item, index) in getItems()"
                :key="String(readRowKeySeed(item) ?? index)"
                class="o-otm-kanban__card"
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
                  <div class="o-otm-kanban__card-title">{{ resolveCardTitle(item) }}</div>
                  <div class="o-otm-kanban__card-meta" v-if="resolveCardSubtitle(item)">{{ resolveCardSubtitle(item) }}</div>
                  <div class="o-otm-kanban__card-actions" v-if="editable || removable" @click.stop>
                    <el-button v-if="editable" size="small" link @click="handleEditItem(index)">{{ _t('Edit') }}</el-button>
                    <el-button v-if="removable" size="small" link type="danger" @click="handleRemoveItem(index)">{{ _t('Delete') }}</el-button>
                  </div>
                </slot>
              </div>
            </template>

            <div v-else class="o-otm-kanban__empty">
              <slot name="empty">{{ effectiveEmptyText }}</slot>
            </div>

            <div v-if="editable" class="o-otm-kanban__add-card" @click="handleAddItem">
              <span>{{ effectiveAddButtonText }}</span>
            </div>
          </div>
        </div>
      </OViewScope>
    </template>

    <template #display>
      <OViewScope view-mode="display" :container="'Kanban'" :field-prefix="String(prop)">
        <div class="o-otm-kanban">
          <div class="o-otm-kanban__board" :style="boardStyle">
            <template v-if="getItems().length > 0">
              <div
                v-for="(item, index) in getItems()"
                :key="String(readRowKeySeed(item) ?? index)"
                class="o-otm-kanban__card"
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
                  <div class="o-otm-kanban__card-title">{{ resolveCardTitle(item) }}</div>
                  <div class="o-otm-kanban__card-meta" v-if="resolveCardSubtitle(item)">{{ resolveCardSubtitle(item) }}</div>
                </slot>
              </div>
            </template>

            <div v-else class="o-otm-kanban__empty">
              <slot name="empty">{{ effectiveEmptyText }}</slot>
            </div>
          </div>
        </div>
      </OViewScope>
    </template>
  </OFieldBase>

  <el-dialog v-model="dialogVisible" :title="dialogTitleText" :width="dialogWidth" append-to-body destroy-on-close>
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
    <div v-else class="o-otm-kanban__dialog-hint">{{ _t('Provide a child record editor via the formView prop.') }}</div>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleDialogCancel">{{ _t('Cancel') }}</el-button>
        <el-button v-if="dialogMode !== 'display'" type="primary" @click="handleDialogSubmit">{{ _t('Save') }}</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, ClientModel<BaseModel>[]>, V = FieldPathType<T, P>">
import { computed, ref, watch, onMounted, provide, useSlots, type Component } from 'vue';
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType, ClientModel } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import type { FormItemProps } from 'element-plus';
import { ElButton, ElDialog, ElMessageBox } from 'element-plus';
import { deepClonePreserve } from '@/core/utils/clone';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import OViewScope from '@/web/web/components/view/OViewScope.vue';
import {
  type OFormChildSubmitApi,
  type OFormChildSubmitApiRegistration,
  type OFormSubmitOutcome,
  type OFormSubmitHandler,
  type OFormSubmitHandlerContext,
} from '@/web/web/components/view/formViewTypes';
import { useField } from '@/web/web/composables/useField';
import type { UseField } from '@/web/web/composables/useField';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OOneToManyKanbanField' });

defineOptions({ name: 'OOneToManyKanbanField', inheritAttrs: false });

type IsAny<TA> = 0 extends 1 & TA ? true : false;
type DialogMode = 'create' | 'edit' | 'display';
const O_FORM_CHILD_SUBMIT_API_REGISTER_KEY = 'o-form-child-submit-api-register';
const O_FORM_EMBEDDED_CONTEXT_KEY = 'o-form-embedded-context';

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
    label: '',
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
const dialogRegisteredFormApis = new Map<string, OFormChildSubmitApi>();
const activeDialogFormToken = ref<string | null>(null);
const dialogFormMode = computed(() => (dialogMode.value === 'display' ? 'display' : 'create'));
const dialogTitleText = computed(() => {
  if (dialogMode.value === 'create') return effectiveCreateDialogTitle.value;
  if (dialogMode.value === 'edit') return effectiveEditDialogTitle.value;
  return effectiveDisplayDialogTitle.value;
});

provide(O_FORM_CHILD_SUBMIT_API_REGISTER_KEY, (registration: OFormChildSubmitApiRegistration) => {
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
provide<boolean>(O_FORM_EMBEDDED_CONTEXT_KEY, true);

function getRegisteredDialogFormApi(): OFormChildSubmitApi | null {
  const token = activeDialogFormToken.value;
  if (!token) return null;
  return dialogRegisteredFormApis.get(token) || null;
}

function readRowKeySeed(row: unknown): string | number | undefined {
  if (!row || typeof row !== 'object') return undefined;
  const r = row as Record<string, any>;
  return r.__rowKey ?? r.Id ?? r.id;
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

  await ElMessageBox.confirm(
    _t('Are you sure you want to delete this record? This action cannot be undone.'),
    _t('Confirm delete'),
    {
      confirmButtonText: _t('Delete'),
      cancelButtonText: _t('Cancel'),
    type: 'warning',
    confirmButtonClass: 'el-button--danger',
  })
    .then(doRemove)
    .catch(() => {});
}

function handleDialogCancel() {
  dialogVisible.value = false;
}

const dialogFormSubmitHandler: OFormSubmitHandler<any> = async (ctx: OFormSubmitHandlerContext<any>) => {
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
    const outcome = submitResult as OFormSubmitOutcome<any>;
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

<style scoped>
.o-otm-kanban {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  min-width: 0;
}

.o-otm-kanban__toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
}

.o-otm-kanban__board {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(var(--kanban-min-card-width), 1fr));
  gap: var(--kanban-gap);
  overflow: auto;
  padding: 2px;
  width: 100%;
  min-width: 0;
}

.o-otm-kanban__card {
  border: 1px solid var(--el-border-color-light);
  border-radius: 8px;
  padding: 12px;
  background: var(--el-fill-color-blank);
  cursor: pointer;
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease;
}

.o-otm-kanban__card:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
}

.o-otm-kanban__card-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
}

.o-otm-kanban__card-meta {
  margin-top: 6px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  word-break: break-word;
}

.o-otm-kanban__card-actions {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.o-otm-kanban__add-card {
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  min-height: 96px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-primary);
  background: color-mix(in oklab, var(--el-color-primary) 5%, transparent);
  cursor: pointer;
  user-select: none;
}

.o-otm-kanban__add-card:hover {
  border-color: var(--el-color-primary);
}

.o-otm-kanban__empty {
  grid-column: 1 / -1;
  min-height: 90px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--el-border-color-light);
  border-radius: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.o-otm-kanban__dialog-hint {
  padding: 18px;
  border: 1px dashed var(--el-border-color);
  border-radius: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.o-otm-kanban__dialog-default-actions {
  margin-top: 10px;
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
