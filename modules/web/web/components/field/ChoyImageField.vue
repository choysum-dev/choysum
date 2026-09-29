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
      <div class="choy-image-field">
        <div v-if="hasAttachment(fieldValue().value)" class="choy-image-current">
          <img v-if="resolvePreviewUrl(fieldValue().value)" class="choy-image-current__preview" :src="resolvePreviewUrl(fieldValue().value)" alt="image preview" />
          <div v-else class="choy-image-current__placeholder" aria-hidden="true">
            <Picture class="size-4" />
          </div>
          <div class="choy-image-current__body">
            <span class="choy-image-current__title" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
            <span v-if="toMetaText(fieldValue().value)" class="choy-image-current__meta">{{ toMetaText(fieldValue().value) }}</span>
            <div class="choy-image-current__actions">
              <label class="choy-image-action-upload">
                <input
                  type="file"
                  class="sr-only"
                  :accept="accept"
                  :multiple="uploadMultiple"
                  :disabled="uploadDisabled"
                  @change="onNativeFileChange($event, fieldValue, onFieldChange)"
                />
                <ChoyButton size="sm" variant="link" class="choy-upload-action-btn" :disabled="uploadDisabled" as="span">{{ replaceButtonText }}</ChoyButton>
              </label>
              <ChoyButton size="sm" variant="destructive" class="choy-upload-action-btn" :disabled="uploadDisabled" @click="removeImage(fieldValue, onFieldChange)">
                {{ _t('Remove') }}
              </ChoyButton>
            </div>
          </div>
        </div>
        <label
          v-else
          class="choy-image-upload"
          :class="{ 'choy-image-upload--drag': uploadDrag }"
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
          <template v-if="uploadDrag">
            <Upload class="choy-upload-drag-icon size-5" />
            <div class="choy-upload-drag-text">{{ uploadDropText }}</div>
          </template>
          <ChoyButton v-else size="sm" variant="link" class="choy-upload-btn" as="span">{{ uploadButtonText }}</ChoyButton>
        </label>
      </div>
    </template>
    <template #display="{ fieldValue, renderMode: slotRenderMode }">
      <div v-if="hasAttachment(fieldValue().value) && isTableRenderMode(slotRenderMode)" class="choy-image-display-row">
        <img
          v-if="resolvePreviewUrl(fieldValue().value)"
          class="choy-image-display-row__preview"
          :src="resolvePreviewUrl(fieldValue().value)"
          alt="image preview"
        />
        <div v-else class="choy-image-display-row__placeholder" aria-hidden="true">
          <span class="inline-flex"><Picture class="size-4" /></span>
        </div>
        <span class="choy-image-display-row__text" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
      </div>
      <a
        v-else-if="hasAttachment(fieldValue().value) && resolveLinkHref(fieldValue().value)"
        class="choy-image-display-card choy-image-display-card--interactive"
        :href="resolveLinkHref(fieldValue().value)"
        target="_blank"
        rel="noopener noreferrer"
        @click.stop
      >
        <img v-if="resolvePreviewUrl(fieldValue().value)" class="choy-image-display-preview" :src="resolvePreviewUrl(fieldValue().value)" alt="image preview" />
        <div v-else class="choy-image-display-placeholder" aria-hidden="true">
          <span class="inline-flex"><Picture class="size-4" /></span>
        </div>
        <span class="choy-image-display-copy">
          <span class="choy-image-display-text" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
          <span v-if="toMetaText(fieldValue().value)" class="choy-image-display-meta">{{ toMetaText(fieldValue().value) }}</span>
        </span>
      </a>
      <div v-else-if="hasAttachment(fieldValue().value)" class="choy-image-display-card">
        <img v-if="resolvePreviewUrl(fieldValue().value)" class="choy-image-display-preview" :src="resolvePreviewUrl(fieldValue().value)" alt="image preview" />
        <div v-else class="choy-image-display-placeholder" aria-hidden="true">
          <span class="inline-flex"><Picture class="size-4" /></span>
        </div>
        <span class="choy-image-display-copy">
          <span class="choy-image-display-text" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
          <span v-if="toMetaText(fieldValue().value)" class="choy-image-display-meta">{{ toMetaText(fieldValue().value) }}</span>
        </span>
      </div>
      <div v-else class="choy-image-display-empty" aria-hidden="true">
        <span class="inline-flex"><Picture /></span>
      </div>
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

<style scoped>
.choy-image-field {
  display: flex;flex-direction: column;align-items: flex-start;gap: 12px;width: min(100%, 360px);max-width: 100%;
}
.choy-image-current,
.choy-image-display-card {
  display: inline-flex;align-items: center;gap: 12px;min-width: 0;max-width: 100%;border: 1px solid var(--choy-color-border);border-radius: 12px;background: var(--choy-color-muted);
}
.choy-image-current {
  width: 100%;padding: 10px 12px;
}
.choy-image-display-card {
  padding: 6px 10px;color: inherit;text-decoration: none;
}
.choy-image-display-card--interactive {
  cursor: pointer;transition:
  border-color 0.2s ease,
  background-color 0.2s ease;
}
.choy-image-display-card--interactive:hover {
  color: inherit;border-color: var(--choy-color-primary-soft);background: var(--choy-color-primary-subtle);
}
.choy-image-display-row {
  display: inline-flex;align-items: center;gap: 8px;min-width: 0;max-width: 124px;
}
.choy-image-display-row__preview,
.choy-image-display-row__placeholder {
  width: 32px;height: 32px;border-radius: 8px;flex: 0 0 auto;
}
.choy-image-display-row__preview {
  display: block;object-fit: cover;border: 1px solid var(--choy-color-border);background: var(--choy-color-background);
}
.choy-image-display-row__placeholder {
  display: inline-flex;align-items: center;justify-content: center;color: var(--choy-color-muted-foreground);background: var(--choy-color-muted);border: 1px dashed var(--choy-color-border);
}
.choy-image-display-row__placeholder :deep(svg) {
  font-size: 16px;
}
.choy-image-display-row__text {
  min-width: 0;color: var(--choy-color-foreground);font-size: 14px;line-height: 1.4;white-space: nowrap;overflow: hidden;text-overflow: ellipsis;
}
.choy-image-current__preview,
.choy-image-current__placeholder {
  width: 48px;height: 48px;border-radius: 10px;flex: 0 0 auto;
}
.choy-image-display-preview,
.choy-image-display-placeholder,
.choy-image-display-empty {
  width: 36px;height: 36px;border-radius: 8px;flex: 0 0 auto;
}
.choy-image-current__preview,
.choy-image-display-preview {
  display: block;object-fit: cover;border: 1px solid var(--choy-color-border);background: var(--choy-color-background);
}
.choy-image-current__placeholder,
.choy-image-display-placeholder,
.choy-image-display-empty {
  display: inline-flex;align-items: center;justify-content: center;color: var(--choy-color-muted-foreground);background: var(--choy-color-muted);border: 1px dashed var(--choy-color-border);
}
.choy-image-current__placeholder :deep(svg),
.choy-image-display-placeholder :deep(svg),
.choy-image-display-empty :deep(svg) {
  font-size: 18px;
}
.choy-image-current__body,
.choy-image-display-copy {
  min-width: 0;display: flex;flex: 1;flex-direction: column;gap: 4px;
}
.choy-image-current__title,
.choy-image-display-text {
  color: var(--choy-color-foreground);font-size: 14px;line-height: 1.4;white-space: nowrap;overflow: hidden;text-overflow: ellipsis;
}
.choy-image-current__meta,
.choy-image-display-meta {
  color: var(--choy-color-muted-foreground);font-size: 12px;line-height: 1.4;white-space: nowrap;overflow: hidden;text-overflow: ellipsis;
}
.choy-image-current__actions {
  display: flex;align-items: center;gap: 12px;flex-wrap: wrap;margin-top: 2px;
}
.choy-image-action-upload {
  display: inline-flex;
}
.choy-image-upload {
  display: block;width: 100%;box-sizing: border-box;
}
.choy-image-upload--drag {
  width: 100%;min-height: 126px;padding: 20px 16px;border-radius: 12px;border: 1px dashed var(--choy-color-border);background: var(--choy-color-muted);cursor: pointer;transition:
  border-color 0.2s ease,
  background-color 0.2s ease;
}
.choy-image-upload--drag:hover {
  border-color: var(--choy-color-primary);background: var(--choy-color-primary-subtle);
}
.choy-upload-drag-icon {
  display: block;margin: 0 auto 10px;font-size: 28px;color: var(--choy-color-primary);
}
.choy-upload-drag-text {
  color: var(--choy-color-muted-foreground);font-size: 13px;line-height: 1.5;text-align: center;
}
.choy-upload-action-btn,
.choy-upload-btn {
  padding: 0;
}
</style>
