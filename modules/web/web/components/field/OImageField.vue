<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OFieldBase
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
      <div class="o-image-field">
        <div v-if="hasAttachment(fieldValue().value)" class="o-image-current">
          <img v-if="resolvePreviewUrl(fieldValue().value)" class="o-image-current__preview" :src="resolvePreviewUrl(fieldValue().value)" alt="image preview" />
          <div v-else class="o-image-current__placeholder" aria-hidden="true">
            <el-icon><Picture /></el-icon>
          </div>
          <div class="o-image-current__body">
            <span class="o-image-current__title" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
            <span v-if="toMetaText(fieldValue().value)" class="o-image-current__meta">{{ toMetaText(fieldValue().value) }}</span>
            <div class="o-image-current__actions">
              <el-upload
                class="o-image-action-upload"
                action="#"
                :auto-upload="false"
                :show-file-list="false"
                :accept="accept"
                :drag="false"
                :multiple="uploadConfig.multiple"
                :limit="uploadConfig.limit"
                :disabled="uploadDisabled"
                :list-type="uploadConfig.listType"
                :on-exceed="uploadConfig.onExceed"
                :on-change="createOnChange(fieldValue, onFieldChange)"
              >
                <el-button link type="primary" class="o-upload-action-btn" :disabled="uploadDisabled">{{ replaceButtonText }}</el-button>
              </el-upload>
              <el-button link type="danger" class="o-upload-action-btn" :disabled="uploadDisabled" @click="removeImage(fieldValue, onFieldChange)"
                >{{ _t('Remove') }}</el-button
              >
            </div>
          </div>
        </div>
        <el-upload
          v-else
          class="o-image-upload"
          action="#"
          :auto-upload="false"
          :show-file-list="shouldShowNativeFileList(fieldValue().value)"
          :accept="accept"
          :drag="shouldUseDragMode(fieldValue().value)"
          :multiple="uploadConfig.multiple"
          :limit="uploadConfig.limit"
          :disabled="uploadDisabled"
          :list-type="uploadConfig.listType"
          :on-exceed="uploadConfig.onExceed"
          :on-change="createOnChange(fieldValue, onFieldChange)"
        >
          <template v-if="uploadDrag">
            <el-icon class="o-upload-drag-icon"><UploadFilled /></el-icon>
            <div class="o-upload-drag-text">{{ uploadDropText }}</div>
          </template>
          <el-button v-else link type="primary" class="o-upload-btn">{{ uploadButtonText }}</el-button>
        </el-upload>
      </div>
    </template>
    <template #display="{ fieldValue, renderMode: slotRenderMode }">
      <div v-if="hasAttachment(fieldValue().value) && isTableRenderMode(slotRenderMode)" class="o-image-display-row">
        <img
          v-if="resolvePreviewUrl(fieldValue().value)"
          class="o-image-display-row__preview"
          :src="resolvePreviewUrl(fieldValue().value)"
          alt="image preview"
        />
        <div v-else class="o-image-display-row__placeholder" aria-hidden="true">
          <el-icon><Picture /></el-icon>
        </div>
        <span class="o-image-display-row__text" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
      </div>
      <a
        v-else-if="hasAttachment(fieldValue().value) && resolveLinkHref(fieldValue().value)"
        class="o-image-display-card o-image-display-card--interactive"
        :href="resolveLinkHref(fieldValue().value)"
        target="_blank"
        rel="noopener noreferrer"
        @click.stop
      >
        <img v-if="resolvePreviewUrl(fieldValue().value)" class="o-image-display-preview" :src="resolvePreviewUrl(fieldValue().value)" alt="image preview" />
        <div v-else class="o-image-display-placeholder" aria-hidden="true">
          <el-icon><Picture /></el-icon>
        </div>
        <span class="o-image-display-copy">
          <span class="o-image-display-text" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
          <span v-if="toMetaText(fieldValue().value)" class="o-image-display-meta">{{ toMetaText(fieldValue().value) }}</span>
        </span>
      </a>
      <div v-else-if="hasAttachment(fieldValue().value)" class="o-image-display-card">
        <img v-if="resolvePreviewUrl(fieldValue().value)" class="o-image-display-preview" :src="resolvePreviewUrl(fieldValue().value)" alt="image preview" />
        <div v-else class="o-image-display-placeholder" aria-hidden="true">
          <el-icon><Picture /></el-icon>
        </div>
        <span class="o-image-display-copy">
          <span class="o-image-display-text" :title="toDisplayText(fieldValue().value)">{{ toDisplayText(fieldValue().value) }}</span>
          <span v-if="toMetaText(fieldValue().value)" class="o-image-display-meta">{{ toMetaText(fieldValue().value) }}</span>
        </span>
      </div>
      <div v-else class="o-image-display-empty" aria-hidden="true">
        <el-icon><Picture /></el-icon>
      </div>
    </template>
  </OFieldBase>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, any>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField, NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import type { UploadFile, UploadRawFile, UploadProps } from 'element-plus';
import { Picture, UploadFilled } from '@element-plus/icons-vue';
import { computed } from 'vue';
import OFieldBase, { type FieldStateExpr } from './OFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';

const { _t } = createTranslate('web', { scope: 'web/components/field/OImageField' });

defineOptions({ name: 'OImageField' });

type IsAny<T> = 0 extends 1 & T ? true : false;
type ViewType = any;
type ValueRefGetter = () => { value: any };
type OnFieldChange = (() => Promise<void>) | undefined;
type UploadPassthroughProps = Partial<Pick<UploadProps, 'drag' | 'multiple' | 'limit' | 'disabled' | 'showFileList' | 'listType' | 'onExceed'>>;
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
    label: '',
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

function normalizeText(value: unknown): string | undefined {
  const text = String(value ?? '').trim();
  return text ? text : undefined;
}

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
  return normalizeText(raw.attachmentBindingId ?? raw.bindingId ?? raw.Id ?? raw.id ?? descriptor?.id);
}

function resolveObjectId(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  return normalizeText(raw.attachmentObjectId ?? raw.objectId);
}

function resolveFileName(raw: unknown): string | undefined {
  if (typeof raw === 'string') return normalizeText(raw);
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeText(raw.fileName ?? raw.displayName ?? raw.name ?? raw.originalFileName ?? descriptor?.fileName);
}

function resolveMimeType(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeText(raw.mimeType ?? raw.contentType ?? raw.clientContentType ?? raw.proposedContentType ?? descriptor?.mimeType);
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
  if (typeof raw === 'string') return !!normalizeText(raw);
  if (isAttachmentObject(raw)) {
    const kind = normalizeText(raw.kind)?.toLowerCase();
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
  return normalizeText(raw.downloadUrl ?? raw.url ?? raw.previewUrl ?? raw.thumbnailUrl ?? descriptor?.downloadUrl ?? descriptor?.previewUrl);
}

function resolvePreviewUrl(raw: unknown): string | undefined {
  if (!isAttachmentObject(raw)) return undefined;
  const descriptor = resolveDescriptor(raw);
  return normalizeText(raw.previewUrl ?? raw.url ?? raw.thumbnailUrl ?? raw.downloadUrl ?? descriptor?.previewUrl ?? descriptor?.downloadUrl);
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
  const valueRef = fieldValue();
  revokeBlobPreview(valueRef.value);
  const previewUrl = await createLocalPreview(file);

  valueRef.value = {
    kind: 'set',
    file,
    fileName: normalizeText(file.name),
    originalFileName: normalizeText(file.name),
    proposedFileName: normalizeText(file.name),
    proposedContentType: normalizeText(file.type),
    clientContentType: normalizeText(file.type),
    previewUrl,
    displayName: normalizeText(file.name),
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

async function removeImage(fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  const valueRef = fieldValue();
  revokeBlobPreview(valueRef.value);
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

<style lang="scss" scoped>
.o-image-field {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  width: min(100%, 360px);
  max-width: 100%;
}

.o-image-current,
.o-image-display-card {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  max-width: 100%;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
}

.o-image-current {
  width: 100%;
  padding: 10px 12px;
}

.o-image-display-card {
  padding: 6px 10px;
  color: inherit;
  text-decoration: none;
}

.o-image-display-card--interactive {
  cursor: pointer;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.o-image-display-card--interactive:hover {
  color: inherit;
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}

.o-image-display-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 124px;
}

.o-image-display-row__preview,
.o-image-display-row__placeholder {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  flex: 0 0 auto;
}

.o-image-display-row__preview {
  display: block;
  object-fit: cover;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}

.o-image-display-row__placeholder {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-placeholder);
  background: var(--el-fill-color-light);
  border: 1px dashed var(--el-border-color);
}

.o-image-display-row__placeholder :deep(svg) {
  font-size: 16px;
}

.o-image-display-row__text {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.o-image-current__preview,
.o-image-current__placeholder {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  flex: 0 0 auto;
}

.o-image-display-preview,
.o-image-display-placeholder,
.o-image-display-empty {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  flex: 0 0 auto;
}

.o-image-current__preview,
.o-image-display-preview {
  display: block;
  object-fit: cover;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}

.o-image-current__placeholder,
.o-image-display-placeholder,
.o-image-display-empty {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-placeholder);
  background: var(--el-fill-color-light);
  border: 1px dashed var(--el-border-color);
}

.o-image-current__placeholder :deep(svg),
.o-image-display-placeholder :deep(svg),
.o-image-display-empty :deep(svg) {
  font-size: 18px;
}

.o-image-current__body,
.o-image-display-copy {
  min-width: 0;
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 4px;
}

.o-image-current__title,
.o-image-display-text {
  color: var(--el-text-color-primary);
  font-size: 14px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.o-image-current__meta,
.o-image-display-meta {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.o-image-current__actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 2px;
}

.o-image-action-upload {
  display: inline-flex;
}

.o-image-action-upload :deep(.el-upload) {
  display: inline-flex;
}

.o-image-upload {
  display: block;
  width: 100%;
}

.o-image-upload :deep(.el-upload) {
  width: 100%;
  display: block;
}

.o-image-upload :deep(.el-upload-dragger) {
  width: 100%;
  min-height: 126px;
  padding: 20px 16px;
  border-radius: 12px;
  background: var(--el-fill-color-lighter);
  border-color: var(--el-border-color);
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease;
}

.o-image-upload :deep(.el-upload-dragger:hover) {
  border-color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.o-upload-drag-icon {
  display: block;
  margin: 0 auto 10px;
  font-size: 28px;
  color: var(--el-color-primary);
}

.o-upload-drag-text {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
  text-align: center;
}

.o-image-upload :deep(.el-upload-list) {
  width: 100%;
  margin: 6px 0 0;
}

.o-upload-action-btn,
.o-upload-btn {
  padding: 0;
}
</style>
