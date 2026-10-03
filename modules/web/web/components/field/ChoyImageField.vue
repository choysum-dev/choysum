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
          class="choy-image-current w-full max-w-full bg-muted"
          state="done"
        >
          <AttachmentMedia variant="image" class="size-12 overflow-hidden border border-border">
            <img
              v-if="resolvePreviewUrl(fieldValue().value)"
              class="choy-image-current__preview size-full object-cover"
              :src="resolvePreviewUrl(fieldValue().value)"
              alt="image preview"
            />
            <Picture v-else class="size-4 text-muted-foreground" />
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle>{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
            <AttachmentDescription v-if="toMetaText(fieldValue().value)">{{ toMetaText(fieldValue().value) }}</AttachmentDescription>
            <AttachmentActions>
              <label class="inline-flex">
                <input
                  type="file"
                  class="sr-only"
                  :accept="accept"
                  :multiple="uploadMultiple"
                  :disabled="uploadDisabled"
                  @change="onNativeFileChange($event, fieldValue, onFieldChange)"
                />
                <ChoyButton size="sm" variant="link" class="p-0" :disabled="uploadDisabled" as="span">{{ replaceButtonText }}</ChoyButton>
              </label>
              <ChoyButton size="sm" variant="destructive" class="p-0" :disabled="uploadDisabled" @click="removeImage(fieldValue, onFieldChange)">
                {{ _t('Remove') }}
              </ChoyButton>
            </AttachmentActions>
          </AttachmentContent>
        </Attachment>
        <label
          v-else
          class="choy-image-upload box-border block w-full cursor-pointer"
          @dragover.prevent="onUploadDragOver"
          @drop.prevent="onNativeFileDrop($event, fieldValue, onFieldChange)"
        >
          <input
            type="file"
            class="sr-only"
            :accept="accept"
            :multiple="uploadMultiple"
            :disabled="uploadDisabled"
            @change="onNativeFileChange($event, fieldValue, onFieldChange)"
          />
          <Attachment
            state="idle"
            class="w-full max-w-full"
            :class="{ 'choy-image-upload--drag min-h-[126px] justify-center px-4 py-5 hover:border-primary hover:bg-primary-subtle': uploadDrag }"
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
        class="max-w-[124px] border-transparent bg-transparent shadow-none"
        state="done"
      >
        <AttachmentMedia variant="image" class="size-8 overflow-hidden border border-border">
          <img
            v-if="resolvePreviewUrl(fieldValue().value)"
            class="size-full object-cover"
            :src="resolvePreviewUrl(fieldValue().value)"
            alt="image preview"
          />
          <Picture v-else class="size-4 text-muted-foreground" />
        </AttachmentMedia>
        <AttachmentContent>
          <AttachmentTitle>{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
        </AttachmentContent>
      </Attachment>
      <a
        v-else-if="hasAttachment(fieldValue().value) && resolveLinkHref(fieldValue().value)"
        class="choy-image-display-card text-inherit no-underline"
        :href="resolveLinkHref(fieldValue().value)"
        target="_blank"
        rel="noopener noreferrer"
        @click.stop
      >
        <Attachment
          class="choy-image-display-card--interactive max-w-full cursor-pointer bg-muted transition-colors hover:border-primary-soft hover:bg-primary-subtle"
          state="done"
        >
          <AttachmentMedia variant="image" class="size-9 overflow-hidden border border-border">
            <img
              v-if="resolvePreviewUrl(fieldValue().value)"
              class="size-full object-cover"
              :src="resolvePreviewUrl(fieldValue().value)"
              alt="image preview"
            />
            <Picture v-else class="size-4 text-muted-foreground" />
          </AttachmentMedia>
          <AttachmentContent>
            <AttachmentTitle>{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
            <AttachmentDescription v-if="toMetaText(fieldValue().value)">{{ toMetaText(fieldValue().value) }}</AttachmentDescription>
          </AttachmentContent>
        </Attachment>
      </a>
      <Attachment
        v-else-if="hasAttachment(fieldValue().value)"
        class="choy-image-display-card max-w-full bg-muted"
        state="done"
      >
        <AttachmentMedia variant="image" class="size-9 overflow-hidden border border-border">
          <img
            v-if="resolvePreviewUrl(fieldValue().value)"
            class="size-full object-cover"
            :src="resolvePreviewUrl(fieldValue().value)"
            alt="image preview"
          />
          <Picture v-else class="size-4 text-muted-foreground" />
        </AttachmentMedia>
        <AttachmentContent>
          <AttachmentTitle>{{ toDisplayText(fieldValue().value) }}</AttachmentTitle>
          <AttachmentDescription v-if="toMetaText(fieldValue().value)">{{ toMetaText(fieldValue().value) }}</AttachmentDescription>
        </AttachmentContent>
      </Attachment>
      <Attachment v-else size="sm" state="idle" class="border-transparent bg-transparent shadow-none">
        <AttachmentMedia variant="icon" class="size-9 border border-dashed border-border bg-muted text-muted-foreground">
          <Picture class="size-4" />
        </AttachmentMedia>
      </Attachment>
    </template>
  </FieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, any>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import { ChoyMessage } from '../../composables/useChoyMessage';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField, NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { computed } from 'vue';
import FieldBase, { type FieldStateExpr } from './FieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import { resolveImageFieldLimitsFromSources, reportImageFieldValidation } from './imageFieldLimits';
import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { Image as Picture, Upload } from 'lucide-vue-next';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import Attachment from '../vendor/ui/attachment/Attachment.vue';
import AttachmentActions from '../vendor/ui/attachment/AttachmentActions.vue';
import AttachmentContent from '../vendor/ui/attachment/AttachmentContent.vue';
import AttachmentDescription from '../vendor/ui/attachment/AttachmentDescription.vue';
import AttachmentMedia from '../vendor/ui/attachment/AttachmentMedia.vue';
import AttachmentTitle from '../vendor/ui/attachment/AttachmentTitle.vue';

const { _t } = createTranslate('web', { scope: 'web/components/field/ImageField' });

defineOptions({ name: 'ChoyImageField' });

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
    accept: 'image/*',
    uploadText: '',
    uploadDropText: '',
    uploadProps: () => ({ drag: true, showFileList: false }),
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;

function resolveFieldLimits() {
  return resolveImageFieldLimitsFromSources({
    bindingProp: binding.prop,
    propsProp: props.prop,
    bindingStore: binding.store,
    propsStore: props.store,
    bindingMeta: binding.meta,
  });
}

const RASTER_IMAGE_TYPES = new Set([
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
  'image/bmp',
  'image/avif',
]);
const RASTER_IMAGE_EXT_RE = /\.(png|jpe?g|gif|webp|bmp|avif)$/i;

function isRasterImage(file: UploadRawFile): boolean {
  const type = String(file.type || '').toLowerCase();
  if (type && type !== 'application/octet-stream') return RASTER_IMAGE_TYPES.has(type);
  return RASTER_IMAGE_EXT_RE.test(String(file.name || ''));
}

async function validateSelectedImageFile(file: UploadRawFile): Promise<boolean> {
  if (!isRasterImage(file)) {
    ChoyMessage.error(_t('Only raster image files are supported.'));
    return false;
  }
  return reportImageFieldValidation(file, resolveFieldLimits(), message => ChoyMessage.error(message));
}

const toView = (raw: any): ViewType => raw;
const fromView = (v: ViewType) => v as unknown as V;
const accept = (props.accept || 'image/*').trim() || 'image/*';
const uploadButtonText = computed(() => (props.uploadText || _t('Upload image')).trim() || _t('Upload image'));
const uploadDropText = computed(() => (props.uploadDropText || _t('Drop image here or click to upload')).trim() || _t('Drop image here or click to upload'));
const uploadConfig = (props.uploadProps || {}) as UploadPassthroughProps;
const uploadMultiple = uploadConfig.multiple ?? false;
const uploadDrag = uploadConfig.drag ?? true;
const uploadShowFileList = uploadConfig.showFileList ?? false;
const uploadDisabled = uploadConfig.disabled ?? false;
const replaceButtonText = computed(() => (uploadMultiple ? uploadButtonText.value : _t('Replace image')));

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

function revokeBlobPreview(raw: unknown): void {
  const preview = resolvePreviewUrl(raw);
  if (!preview || !preview.startsWith('blob:')) return;
  if (typeof URL === 'undefined' || typeof URL.revokeObjectURL !== 'function') return;
  try {
    URL.revokeObjectURL(preview);
  } catch {}
}

async function createLocalPreview(file: Blob): Promise<string | undefined> {
  if (typeof FileReader === 'undefined') return undefined;
  return await new Promise(resolve => {
    const reader = new FileReader();
    reader.onload = () => {
      resolve(typeof reader.result === 'string' ? reader.result : undefined);
    };
    reader.onerror = () => resolve(undefined);
    reader.onabort = () => resolve(undefined);
    try {
      reader.readAsDataURL(file);
    } catch {
      resolve(undefined);
    }
  });
}

function hasAttachment(raw: unknown): boolean {
  if (raw == null) return false;
  if (typeof raw === 'string') return !!normalizeOptionalString(raw);
  if (isAttachmentObject(raw)) {
    const kind = normalizeOptionalString(raw.kind)?.toLowerCase();
    if (kind === 'noop' || kind === 'clear') return false;
    return !!(resolveBindingId(raw) || resolveFileName(raw) || resolveObjectId(raw) || resolvePreviewUrl(raw) || resolveDownloadUrl(raw) || raw.file);
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

function resolveDownloadUrl(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  const fromDownloadUrl = normalizeOptionalString(raw.downloadUrl);
  if (fromDownloadUrl) return fromDownloadUrl;
  const fromUrl = normalizeOptionalString(raw.url);
  if (fromUrl) return fromUrl;
  const fromPreviewUrl = normalizeOptionalString(raw.previewUrl);
  if (fromPreviewUrl) return fromPreviewUrl;
  const fromThumbnailUrl = normalizeOptionalString(raw.thumbnailUrl);
  if (fromThumbnailUrl) return fromThumbnailUrl;
  const fromDescriptorDownload = normalizeOptionalString(descriptor?.downloadUrl);
  if (fromDescriptorDownload) return fromDescriptorDownload;
  return normalizeOptionalString(descriptor?.previewUrl);
}

function resolvePreviewUrl(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  const fromPreviewUrl = normalizeOptionalString(raw.previewUrl);
  if (fromPreviewUrl) return fromPreviewUrl;
  const fromUrl = normalizeOptionalString(raw.url);
  if (fromUrl) return fromUrl;
  const fromThumbnailUrl = normalizeOptionalString(raw.thumbnailUrl);
  if (fromThumbnailUrl) return fromThumbnailUrl;
  const fromDownloadUrl = normalizeOptionalString(raw.downloadUrl);
  if (fromDownloadUrl) return fromDownloadUrl;
  const fromDescriptorPreview = normalizeOptionalString(descriptor?.previewUrl);
  if (fromDescriptorPreview) return fromDescriptorPreview;
  return normalizeOptionalString(descriptor?.downloadUrl);
}

function resolveLinkHref(raw: unknown): string | undefined {
  return resolveDownloadUrl(raw) ?? resolvePreviewUrl(raw);
}

function isTableRenderMode(renderMode: unknown): boolean {
  return renderMode === 'table';
}

function toMetaText(raw: unknown): string | undefined {
  const mimeType = resolveMimeType(raw);
  const sizeText = formatSize(resolveSizeBytes(raw));
  const summary = [mimeType, sizeText].filter(Boolean).join(' · ');
  return summary || undefined;
}

async function applySelectedImage(file: UploadRawFile, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  if (!(await validateSelectedImageFile(file))) {
    return;
  }
  const valueRef = fieldValue();
  // Local previews are data: URLs; any blob: URL in the value is host-owned and must not be revoked here.
  const previewUrl = await createLocalPreview(file);
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
    previewUrl,
    displayName: fileName,
  };

  if (typeof onFieldChange === 'function') {
    await onFieldChange();
  }
}

async function onUploadChange(uploadFile: UploadFile, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  const rawFile = uploadFile.raw;
  if (!rawFile) return;
  await applySelectedImage(rawFile, fieldValue, onFieldChange);
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
  await applySelectedImage(file, fieldValue, onFieldChange);
  input.value = '';
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
  await applySelectedImage(file, fieldValue, onFieldChange);
}

async function removeImage(fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  const valueRef = fieldValue();
  // Do not revoke blob: previews — createLocalPreview uses data: URLs; blob: values are host-owned.
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
    return '[image]';
  }
  return String(raw);
}
</script>

