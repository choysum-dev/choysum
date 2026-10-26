<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    :binding="binding"
    :label="label"
    :rules="mergedRules"
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
    v-bind="$attrs"
  >
    <template #edit="{ fieldValue, inputId, inputName, describedBy, invalid, onFieldChange }">
      <ChoyAttachmentFieldChrome
        mode="edit"
        class="choy-binary-field"
        compact
        current-class="choy-binary-current"
        upload-class="choy-binary-upload"
        :has-value="hasAttachment(fieldValue().value)"
        :title="toDisplayText(fieldValue().value)"
        :meta="toAttachmentCompactMetaText(fieldValue().value)"
        :tooltip="toAttachmentTooltipText(fieldValue().value)"
        :open-href="resolveDownloadUrl(fieldValue().value)"
        :download-file-name="resolveFileName(fieldValue().value)"
        :input-id="inputId"
        :input-name="inputName"
        :described-by="describedBy"
        :invalid="invalid"
        :accept="accept"
        :drag="drag"
        :disabled="disabled"
        :busy="isUploadBusy(fieldValue().value)"
        :attachment-state="editAttachmentState(fieldValue().value)"
        :upload-label="uploadButtonText"
        :drop-label="uploadDropLabel"
        :replace-label="replaceButtonText"
        :remove-label="_t('Remove')"
        :open-label="_t('Download')"
        @file="file => onPickedFile(file, fieldValue, onFieldChange)"
        @remove="removeBinary(fieldValue, onFieldChange)"
      >
        <template #media>
          <AttachmentMedia
            variant="icon"
            aria-hidden="true"
            class="border-0 bg-transparent text-muted-foreground shadow-none"
            :data-binary-icon="resolveBinaryMediaIconKind(fieldValue().value)"
          >
            <component :is="binaryMediaIcon(fieldValue().value)" class="size-4" />
          </AttachmentMedia>
        </template>
      </ChoyAttachmentFieldChrome>
    </template>
    <template #display="{ fieldValue, renderMode: slotRenderMode }">
      <ChoyAttachmentFieldChrome
        mode="display"
        class="choy-binary-field"
        compact
        :has-value="hasAttachment(fieldValue().value)"
        :title="toDisplayText(fieldValue().value)"
        :meta="isTableRenderMode(slotRenderMode) ? undefined : toAttachmentCompactMetaText(fieldValue().value)"
        :tooltip="toAttachmentTooltipText(fieldValue().value)"
        :open-href="resolveDownloadUrl(fieldValue().value)"
        :download-file-name="resolveFileName(fieldValue().value)"
        :table="isTableRenderMode(slotRenderMode)"
        table-class="max-w-36"
        :upload-label="uploadButtonText"
        :drop-label="uploadDropLabel"
        :replace-label="replaceButtonText"
        :remove-label="_t('Remove')"
        :open-label="_t('Download')"
      >
        <template #media>
          <AttachmentMedia
            variant="icon"
            aria-hidden="true"
            class="border-0 bg-transparent text-muted-foreground shadow-none"
            :data-binary-icon="resolveBinaryMediaIconKind(fieldValue().value)"
          >
            <component :is="binaryMediaIcon(fieldValue().value)" class="size-4" />
          </AttachmentMedia>
        </template>
      </ChoyAttachmentFieldChrome>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, any>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField, NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import { computed, inject, ref, type Ref } from 'vue';
import { normalizeOptionalString } from '@/core/service/utils/normalization';
import type { Component } from 'vue';
import {
  File,
  FileArchive,
  FileAudio,
  FileCode,
  FileSpreadsheet,
  FileText,
  FileVideo,
  Image,
} from 'lucide-vue-next';
import AttachmentMedia from '../vendor/ui/attachment/AttachmentMedia.vue';
import ChoyAttachmentFieldChrome from './ChoyAttachmentFieldChrome.vue';
import {
  hasAttachment,
  hasPendingLocalFile,
  isAcceptAllowed,
  isTableRenderMode,
  resolveBinaryMediaIconKind,
  resolveDownloadUrl,
  resolveFileName,
  toAttachmentCompactMetaText,
  toAttachmentDisplayText,
  toAttachmentTooltipText,
  type BinaryMediaIconKind,
} from './attachmentFieldHelpers';

const BINARY_MEDIA_ICONS: Record<BinaryMediaIconKind, Component> = {
  image: Image,
  pdf: FileText,
  spreadsheet: FileSpreadsheet,
  archive: FileArchive,
  video: FileVideo,
  audio: FileAudio,
  code: FileCode,
  file: File,
};

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyBinaryField' });

defineOptions({ name: 'ChoyBinaryField' });

type IsAny<T> = 0 extends 1 & T ? true : false;
type ViewType = any;
type ValueRefGetter = () => { value: any };
type OnFieldChange = (() => Promise<void>) | undefined;

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
    vColumnProps?: Record<string, unknown>;
    agg?: NarrowAggProp<NonNumericAggFns>;
    accept?: string;
    uploadText?: string;
    uploadDropText?: string;
    /** Show dashed drop zone when empty (default true). */
    drag?: boolean;
    disabled?: boolean;
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
    vColumnProps: () => ({}),
    accept: '',
    uploadText: '',
    uploadDropText: '',
    drag: true,
    disabled: false,
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;
const fieldErrors = inject<Ref<Map<string, string>> | null>('field-errors', null);
const formSubmitting = inject<Ref<boolean> | undefined>('form-submitting', undefined);

const pickError = ref('');

const pickErrorRule = {
  validator: (_r: unknown, _value: unknown, cb: (error?: Error) => void) => {
    if (pickError.value) return cb(new Error(pickError.value));
    cb();
  },
} as RuleItem;

const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), pickErrorRule]);

const toView = (raw: any): ViewType => raw;
const fromView = (v: ViewType) => v as unknown as V;
const accept = computed(() => (props.accept || '').trim());
const uploadButtonText = computed(() => (props.uploadText || _t('Upload file')).trim() || _t('Upload file'));
const uploadDropLabel = computed(
  () => (props.uploadDropText || _t('Drop file here or click to upload')).trim() || _t('Drop file here or click to upload')
);
const replaceButtonText = computed(() => _t('Replace file'));

function toDisplayText(raw: any): string {
  return toAttachmentDisplayText(raw, '[binary]');
}

function binaryMediaIcon(raw: unknown): Component {
  return BINARY_MEDIA_ICONS[resolveBinaryMediaIconKind(raw)] ?? File;
}

function fieldPropKey(): string {
  return String(binding.prop || props.prop || '');
}

function setPickError(message: string): void {
  pickError.value = message;
  const key = fieldPropKey();
  const map = fieldErrors?.value;
  if (map && key) {
    map.set(key, message);
  }
}

function clearPickError(): void {
  pickError.value = '';
  const key = fieldPropKey();
  const map = fieldErrors?.value;
  if (map && key) {
    map.delete(key);
  }
}

function isUploadBusy(raw: unknown): boolean {
  return !!formSubmitting?.value && hasPendingLocalFile(raw);
}

function editAttachmentState(raw: unknown): 'idle' | 'uploading' | 'error' | 'done' {
  if (isUploadBusy(raw)) return 'uploading';
  if (pickError.value && !hasAttachment(raw)) return 'error';
  if (hasAttachment(raw)) return 'done';
  return 'idle';
}

async function applySelectedBinary(file: File, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  if (!isAcceptAllowed(file, accept.value)) {
    setPickError(_t('Selected file type is not allowed.'));
    return;
  }
  clearPickError();
  const valueRef = fieldValue();
  const fileName = normalizeOptionalString(file.name);
  const contentType = normalizeOptionalString(file.type);

  valueRef.value = {
    kind: 'set',
    file,
    fileName,
    originalFileName: fileName,
    proposedFileName: fileName,
    proposedContentType: contentType,
    clientContentType: contentType,
    displayName: fileName,
  };

  if (typeof onFieldChange === 'function') {
    await onFieldChange();
  }
}

async function onPickedFile(file: File, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  await applySelectedBinary(file, fieldValue, onFieldChange);
}

async function removeBinary(fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  clearPickError();
  const valueRef = fieldValue();
  valueRef.value = null;
  if (typeof onFieldChange === 'function') {
    await onFieldChange();
  }
}
</script>
