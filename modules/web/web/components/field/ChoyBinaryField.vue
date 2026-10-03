<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <FieldBase
    :binding="binding"
    :label="label"
    :rules="rules"
    :formItemProps="formItemProps"
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
    <template #edit="{ fieldValue, onFieldChange }">
      <div class="flex w-full max-w-[360px] flex-col items-start gap-3">
        <Attachment
          v-if="hasAttachment(fieldValue().value)"
          class="choy-binary-current w-full max-w-full bg-muted"
          state="done"
        >
          <AttachmentMedia variant="icon" aria-hidden="true" class="border border-primary-muted bg-primary-subtle text-primary">
            <FileText class="size-4" />
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
            <AttachmentDescription v-if="toMetaText(fieldValue().value)">{{ toMetaText(fieldValue().value) }}</AttachmentDescription>
            <AttachmentActions>
              <label class="inline-flex">
                <input
                  type="file"
                  class="sr-only"
                  :accept="accept || undefined"
                  :multiple="uploadMultiple"
                  :disabled="uploadDisabled"
                  @change="onNativeFileChange($event, fieldValue, onFieldChange)"
                />
                <ChoyButton size="sm" variant="link" class="p-0" :disabled="uploadDisabled" as="span">{{ replaceButtonText }}</ChoyButton>
              </label>
              <ChoyButton size="sm" variant="destructive" class="p-0" :disabled="uploadDisabled" @click="removeBinary(fieldValue, onFieldChange)">
                {{ _t('Remove') }}
              </ChoyButton>
            </AttachmentActions>
          </AttachmentContent>
        </Attachment>
        <label
          v-else
          class="choy-binary-upload box-border block w-full cursor-pointer"
          @dragover.prevent="onUploadDragOver"
          @drop.prevent="onNativeFileDrop($event, fieldValue, onFieldChange)"
        >
          <input
            type="file"
            class="sr-only"
            :accept="accept || undefined"
            :multiple="uploadMultiple"
            :disabled="uploadDisabled"
            @change="onNativeFileChange($event, fieldValue, onFieldChange)"
          />
          <Attachment
            state="idle"
            class="w-full max-w-full"
            :class="{ 'choy-binary-upload--drag min-h-[126px] justify-center px-4 py-5 hover:border-primary hover:bg-primary-subtle': uploadDrag }"
          >
            <template v-if="uploadDrag">
              <Upload class="mx-auto mb-2.5 block size-5 text-primary" />
              <div class="w-full text-center text-[13px] leading-normal text-muted-foreground">{{ uploadDropText }}</div>
            </template>
            <ChoyButton v-else size="sm" variant="link" class="p-0" as="span">{{ uploadButtonText }}</ChoyButton>
          </Attachment>
        </label>
      </div>
    </template>
    <template #display="{ fieldValue, renderMode: slotRenderMode }">
      <Attachment
        v-if="hasAttachment(fieldValue().value) && isTableRenderMode(slotRenderMode)"
        size="sm"
        class="max-w-[140px] border-transparent bg-transparent shadow-none"
        state="done"
      >
        <AttachmentMedia variant="icon" aria-hidden="true" class="size-[30px] border border-primary-muted bg-primary-subtle text-primary">
          <FileText class="size-4" />
        </AttachmentMedia>
        <AttachmentContent>
          <AttachmentTitle :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
        </AttachmentContent>
      </Attachment>
      <a
        v-else-if="hasAttachment(fieldValue().value) && resolveDownloadUrl(fieldValue().value)"
        class="choy-binary-display-card text-inherit no-underline"
        :href="resolveDownloadUrl(fieldValue().value)"
        target="_blank"
        rel="noopener noreferrer"
        @click.stop
      >
        <Attachment
          class="choy-binary-display-card--interactive max-w-full cursor-pointer bg-muted transition-colors hover:border-primary-soft hover:bg-primary-subtle"
          state="done"
        >
          <AttachmentMedia variant="icon" aria-hidden="true" class="border border-primary-muted bg-primary-subtle text-primary">
            <FileText class="size-4" />
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
            <AttachmentDescription v-if="toMetaText(fieldValue().value)">{{ toMetaText(fieldValue().value) }}</AttachmentDescription>
          </AttachmentContent>
        </Attachment>
      </a>
      <Attachment
        v-else-if="hasAttachment(fieldValue().value)"
        class="choy-binary-display-card max-w-full bg-muted"
        state="done"
      >
        <AttachmentMedia variant="icon" aria-hidden="true" class="border border-primary-muted bg-primary-subtle text-primary">
          <FileText class="size-4" />
        </AttachmentMedia>
        <AttachmentContent>
          <AttachmentTitle :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
          <AttachmentDescription v-if="toMetaText(fieldValue().value)">{{ toMetaText(fieldValue().value) }}</AttachmentDescription>
        </AttachmentContent>
      </Attachment>
      <span v-else class="choy-field-display-empty inline-flex min-h-[34px] items-center text-muted-foreground">-</span>
    </template>
  </FieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, any>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField, NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import FieldBase, { type FieldStateExpr } from './FieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import { computed } from 'vue';
import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { FileText, Upload } from 'lucide-vue-next';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { ChoyMessage } from '../../composables/useChoyMessage';
import Attachment from '../vendor/ui/attachment/Attachment.vue';
import AttachmentActions from '../vendor/ui/attachment/AttachmentActions.vue';
import AttachmentContent from '../vendor/ui/attachment/AttachmentContent.vue';
import AttachmentDescription from '../vendor/ui/attachment/AttachmentDescription.vue';
import AttachmentMedia from '../vendor/ui/attachment/AttachmentMedia.vue';
import AttachmentTitle from '../vendor/ui/attachment/AttachmentTitle.vue';

const { _t } = createTranslate('web', { scope: 'web/components/field/BinaryField' });

defineOptions({ name: 'ChoyBinaryField' });

type IsAny<T> = 0 extends 1 & T ? true : false;
type ViewType = any;
type ValueRefGetter = () => { value: any };
type OnFieldChange = (() => Promise<void>) | undefined;
type UploadPassthroughProps = {
  drag?: boolean;
  multiple?: boolean;
  limit?: number;
  disabled?: boolean;
  showFileList?: boolean;
  listType?: string;
  onExceed?: (...args: any[]) => void;
};
type UploadRawFile = File;
type UploadFile = { raw?: UploadRawFile | null; name?: string };
type AttachmentLike = Record<string, any>;

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

    formItemProps?: Record<string, unknown>;
    vColumnProps?: Record<string, unknown>;
    agg?: NarrowAggProp<NonNumericAggFns>;
    accept?: string;
    uploadText?: string;
    uploadDropText?: string;
    uploadProps?: UploadPassthroughProps;
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
    uploadProps: () => ({ drag: true, showFileList: false }),
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

const toView = (raw: any): ViewType => raw;
const fromView = (v: ViewType) => v as unknown as V;
const accept = (props.accept || '').trim();
const uploadButtonText = computed(() => (props.uploadText || _t('Upload file')).trim() || _t('Upload file'));
const uploadDropText = computed(() => (props.uploadDropText || _t('Drop file here or click to upload')).trim() || _t('Drop file here or click to upload'));
const uploadConfig = (props.uploadProps || {}) as UploadPassthroughProps;
const uploadMultiple = uploadConfig.multiple ?? false;
const uploadDrag = uploadConfig.drag ?? true;
const uploadShowFileList = uploadConfig.showFileList ?? false;
const uploadDisabled = uploadConfig.disabled ?? false;
const replaceButtonText = computed(() => (uploadMultiple ? uploadButtonText.value : _t('Replace file')));

function isAttachmentObject(raw: unknown): raw is AttachmentLike {
  return !!raw && typeof raw === 'object' && !Array.isArray(raw);
}

function resolveDescriptor(raw: unknown): AttachmentLike | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = raw.descriptor;
  return descriptor && typeof descriptor === 'object' && !Array.isArray(descriptor) ? (descriptor as AttachmentLike) : undefined;
}

function resolveBindingId(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeOptionalString(raw.attachmentBindingId ?? raw.bindingId ?? raw.Id ?? descriptor?.id);
}

function resolveObjectId(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const fromAttachmentObjectId = normalizeOptionalString(raw.attachmentObjectId);
  if (fromAttachmentObjectId) return fromAttachmentObjectId;
  return normalizeOptionalString(raw.objectId);
}

function resolveFileName(raw: unknown): string | undefined {
  if (typeof raw === 'string') return normalizeOptionalString(raw);
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeOptionalString(raw.fileName ?? raw.displayName ?? raw.name ?? raw.originalFileName ?? descriptor?.fileName);
}

function resolveMimeType(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeOptionalString(raw.mimeType ?? raw.contentType ?? raw.clientContentType ?? raw.proposedContentType ?? descriptor?.mimeType);
}

function resolveSizeBytes(raw: unknown): number | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  const source = raw.sizeBytes ?? raw.size ?? raw.file?.size ?? descriptor?.sizeBytes;
  const size = typeof source === 'number' ? source : Number(source);
  return Number.isFinite(size) && size >= 0 ? size : undefined;
}

function resolveDownloadUrl(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  const fromDownloadUrl = normalizeOptionalString(raw.downloadUrl);
  if (fromDownloadUrl) return fromDownloadUrl;
  const fromUrl = normalizeOptionalString(raw.url);
  if (fromUrl) return fromUrl;
  const fromPreviewUrl = normalizeOptionalString(raw.previewUrl);
  if (fromPreviewUrl) return fromPreviewUrl;
  const fromDescriptorDownload = normalizeOptionalString(descriptor?.downloadUrl);
  if (fromDescriptorDownload) return fromDescriptorDownload;
  return normalizeOptionalString(descriptor?.previewUrl);
}

function hasAttachment(raw: unknown): boolean {
  if (raw == null) return false;
  if (typeof raw === 'string') return !!normalizeOptionalString(raw);
  if (isAttachmentObject(raw)) {
    const kind = normalizeOptionalString(raw.kind)?.toLowerCase();
    if (kind === 'noop' || kind === 'clear') return false;
    return !!(resolveBindingId(raw) || resolveFileName(raw) || resolveObjectId(raw) || resolveDownloadUrl(raw) || raw.file);
  }
  return true;
}

function shouldShowUploadTrigger(raw: unknown): boolean {
  if (uploadMultiple) return true;
  return !hasAttachment(raw);
}

function shouldUseDragMode(raw: unknown): boolean {
  return uploadDrag && shouldShowUploadTrigger(raw);
}

function shouldShowNativeFileList(raw: unknown): boolean {
  return uploadShowFileList && shouldShowUploadTrigger(raw);
}

function formatSize(sizeBytes: number | undefined): string | undefined {
  if (sizeBytes == null) return undefined;
  if (sizeBytes < 1024) return `${sizeBytes} B`;
  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = sizeBytes / 1024;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const digits = value >= 10 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

function toMetaText(raw: unknown): string | undefined {
  const mimeType = resolveMimeType(raw);
  const sizeText = formatSize(resolveSizeBytes(raw));
  const summary = [mimeType, sizeText].filter(Boolean).join(' · ');
  return summary || undefined;
}

function isTableRenderMode(renderMode: unknown): boolean {
  return renderMode === 'table';
}

/** Enforce accept tokens beyond the native input hint (drag/drop and *.ext). */
function isAcceptAllowed(file: File): boolean {
  const name = String(file.name || '').toLowerCase();
  const type = String(file.type || '').toLowerCase();
  const tokens = accept.split(',').map(t => t.trim().toLowerCase()).filter(Boolean);
  const restrictive = tokens.filter(t => t !== '*' && t !== '*/*');
  if (!restrictive.length) return true;
  return restrictive.some(token => {
    if (token.startsWith('*.')) return name.endsWith(token.slice(1));
    if (token.startsWith('.')) return name.endsWith(token);
    if (token.endsWith('/*')) return type !== '' && type.startsWith(token.slice(0, -1));
    return type === token;
  });
}

async function applySelectedBinary(file: UploadRawFile, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  if (!isAcceptAllowed(file)) {
    ChoyMessage.error(_t('Selected file type is not allowed.'));
    return;
  }
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

async function onUploadChange(uploadFile: UploadFile, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  const rawFile = uploadFile.raw;
  if (!rawFile) return;
  await applySelectedBinary(rawFile, fieldValue, onFieldChange);
}

function createOnChange(fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange) {
  return async (uploadFile: UploadFile): Promise<void> => {
    await onUploadChange(uploadFile, fieldValue, onFieldChange);
  };
}

async function onNativeFileChange(ev: Event, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  input.value = '';
  await applySelectedBinary(file, fieldValue, onFieldChange);
}

function onUploadDragOver(ev: DragEvent): void {
  if (uploadDisabled) return;
  ev.dataTransfer && (ev.dataTransfer.dropEffect = 'copy');
}

async function onNativeFileDrop(
  ev: DragEvent,
  fieldValue: ValueRefGetter,
  onFieldChange?: OnFieldChange,
): Promise<void> {
  if (uploadDisabled) return;
  const file = ev.dataTransfer?.files?.[0];
  if (!file) return;
  await applySelectedBinary(file, fieldValue, onFieldChange);
}

async function removeBinary(fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  const valueRef = fieldValue();
  valueRef.value = null;
  if (typeof onFieldChange === 'function') {
    await onFieldChange();
  }
}

function toDisplayText(raw: any): string {
  if (raw == null) return '';
  const fileName = resolveFileName(raw);
  if (fileName) return fileName;
  const bindingId = resolveBindingId(raw);
  if (bindingId) return bindingId;
  if (typeof raw === 'object' && raw && !Array.isArray(raw)) {
    return '[binary]';
  }
  return String(raw);
}
</script>

