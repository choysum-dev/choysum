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
        class="choy-image-field"
        presentation="image"
        image-click-action="download"
        current-class="choy-image-current"
        upload-class="choy-image-upload"
        :width="width"
        :height="height"
        :fit="fit"
        :radius="radius"
        :natural-width="thumbNatural?.width"
        :natural-height="thumbNatural?.height"
        :has-value="hasAttachment(fieldValue().value)"
        :title="toDisplayText(fieldValue().value)"
        :open-href="resolveLinkHref(fieldValue().value)"
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
        @remove="removeImage(fieldValue, onFieldChange)"
      >
        <template #media>
          <AttachmentMedia variant="image" class="choy-image-avatar__media size-full overflow-hidden">
            <img
              v-if="previewSrc(fieldValue().value)"
              class="choy-image-current__preview size-full"
              :class="fit === 'cover' ? 'object-cover' : 'object-contain'"
              :src="previewSrc(fieldValue().value)"
              :alt="previewAlt(fieldValue().value)"
              @load="onThumbLoad"
              @error="onPreviewError(fieldValue().value)"
            />
            <Picture v-else class="size-6 text-muted-foreground" />
          </AttachmentMedia>
        </template>
      </ChoyAttachmentFieldChrome>
    </template>
    <template #display="{ fieldValue, renderMode: slotRenderMode }">
      <ChoyAttachmentFieldChrome
        mode="display"
        class="choy-image-field"
        :presentation="isTableRenderMode(slotRenderMode) ? 'file' : 'image'"
        :image-click-action="isTableRenderMode(slotRenderMode) ? 'download' : 'preview'"
        :width="width"
        :height="height"
        :fit="isTableRenderMode(slotRenderMode) ? 'cover' : fit"
        :radius="isTableRenderMode(slotRenderMode) ? 'md' : radius"
        :natural-width="isTableRenderMode(slotRenderMode) ? undefined : thumbNatural?.width"
        :natural-height="isTableRenderMode(slotRenderMode) ? undefined : thumbNatural?.height"
        :has-value="hasAttachment(fieldValue().value)"
        :title="toDisplayText(fieldValue().value)"
        :meta="isTableRenderMode(slotRenderMode) ? toAttachmentMetaText(fieldValue().value) : undefined"
        :open-href="resolveLinkHref(fieldValue().value)"
        :download-file-name="resolveFileName(fieldValue().value)"
        :table="isTableRenderMode(slotRenderMode)"
        table-class="max-w-32"
        :upload-label="uploadButtonText"
        :drop-label="uploadDropLabel"
        :replace-label="replaceButtonText"
        :remove-label="_t('Remove')"
        :open-label="_t('Download')"
        :preview-label="_t('Image preview')"
        @preview="openImagePreview(fieldValue().value)"
      >
        <template #media>
          <AttachmentMedia
            variant="image"
            :class="
              isTableRenderMode(slotRenderMode)
                ? 'size-8 overflow-hidden rounded-md border border-border'
                : 'choy-image-avatar__media size-full overflow-hidden'
            "
          >
            <img
              v-if="previewSrc(fieldValue().value)"
              class="choy-image-current__preview size-full"
              :class="isTableRenderMode(slotRenderMode) || fit === 'cover' ? 'object-cover' : 'object-contain'"
              :src="previewSrc(fieldValue().value)"
              :alt="previewAlt(fieldValue().value)"
              @load="onThumbLoad"
              @error="onPreviewError(fieldValue().value)"
            />
            <Picture v-else class="size-4 text-muted-foreground" />
          </AttachmentMedia>
        </template>
      </ChoyAttachmentFieldChrome>
    </template>
  </ChoyFieldBase>

  <ChoyDialog v-model:open="previewDialogOpen">
    <ChoyDialogContent
      class="choy-image-preview-dialog !block !h-fit !w-fit !max-w-none !gap-0 !border-0 !bg-transparent !p-0 !shadow-none"
      :aria-label="previewDialogTitle"
      @open-auto-focus.prevent
    >
      <!--
        Size the inner shell: DialogContent portals to body and does not forward
        style attrs onto the panel, so parent scoped CSS / :style on Content are unreliable.
      -->
      <div
        class="choy-image-preview-dialog__shell relative overflow-hidden rounded-lg border border-border bg-background shadow-lg"
        :style="previewDialogShellStyle"
      >
        <img
          v-if="previewDialogSrc"
          ref="previewImgRef"
          class="choy-image-preview-dialog__img block"
          :src="previewDialogSrc"
          :alt="previewDialogAlt"
          :style="previewDialogImgStyle"
          @load="onPreviewDialogImgLoad"
        />
      </div>
    </ChoyDialogContent>
  </ChoyDialog>
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, any>, V = FieldPathType<T, P>">
import type { RuleItem } from 'async-validator';
import type { BaseModel, FieldPath, FieldPathType } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useField } from '@/web/web/composables/useField';
import type { UseField, NarrowAggProp, NonNumericAggFns } from '@/web/web/composables/useField';
import { computed, inject, nextTick, ref, watch, type Ref } from 'vue';
import ChoyFieldBase, { type FieldStateExpr, type FormItemProps } from './ChoyFieldBase.vue';
import { createTranslate } from '@/web/web/i18n';
import {
  resolveImageFieldLimitsFromSources,
  validateImageFieldFile,
  imageFieldLimitErrorMessage,
} from './imageFieldLimits';
import { normalizeOptionalString } from '@/core/service/utils/normalization';
import { Image as Picture } from 'lucide-vue-next';
import AttachmentMedia from '../vendor/ui/attachment/AttachmentMedia.vue';
import ChoyAttachmentFieldChrome from './ChoyAttachmentFieldChrome.vue';
import { ChoyDialog, ChoyDialogContent } from '@/web/web/components/layout/choyDialog';
import {
  hasAttachment,
  hasPendingLocalFile,
  isTableRenderMode,
  resolveDownloadUrl,
  resolveFileName,
  resolvePreviewUrl,
  toAttachmentDisplayText,
  toAttachmentMetaText,
  type ImageAvatarFit,
  type ImageAvatarRadius,
} from './attachmentFieldHelpers';
import {
  fetchAuthenticatedPreviewAsDataUrl,
  isInternalDocumentBindingContentUrl,
} from './attachmentPreviewAuth';
import { resolveImagePreviewDialogSize } from './imagePreviewDialogSize';

const { _t } = createTranslate('web', { scope: 'web/components/field/ChoyImageField' });

defineOptions({ name: 'ChoyImageField' });

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
    /** Form avatar box width (px number or CSS length). Default 112. For contain, max width. */
    width?: number | string;
    /** Form avatar box height (px number or CSS length). For contain, max height. */
    height?: number | string;
    /**
     * Form thumbnail fit. Default contain (preserve ratio). Table cells always use cover.
     * cover = crop to fixed box (avatar-style).
     */
    fit?: ImageAvatarFit;
    /** Form thumbnail corner radius. Default md. Table cells use md. */
    radius?: ImageAvatarRadius;
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
    drag: true,
    disabled: false,
    width: undefined,
    height: undefined,
    fit: 'contain',
    radius: 'md',
    renderMode: 'auto',
    showInlineError: false,
  }
);

const binding = (props.binding ?? useField<T, P, V>({ store: props.store as WebModelStore<T>, prop: props.prop as P, agg: props.agg })) as UseField<T, V>;
const fieldErrors = inject<Ref<Map<string, string>> | null>('field-errors', null);
const formSubmitting = inject<Ref<boolean> | undefined>('form-submitting', undefined);

const pickError = ref('');
/** Preview URLs that failed to load — fall back to Picture icon. */
const brokenPreviewUrls = ref(new Set<string>());
/** Source preview URL → authenticated data: URL (CSP-safe for <img>). */
const authPreviewBySource = ref(new Map<string, string>());
/** Internal URLs that already failed authenticated fetch — do not retry. */
const authPreviewFailed = ref(new Set<string>());

/** Form thumbnail intrinsic size for contain layout. */
const thumbNatural = ref<{ width: number; height: number } | null>(null);

const previewDialogOpen = ref(false);
const previewDialogSrc = ref('');
const previewDialogAlt = ref('');
const previewDialogNatural = ref<{ width: number; height: number } | null>(null);
const previewImgRef = ref<HTMLImageElement | null>(null);
/** DialogContent default padding (!p-2) on each side. */
const PREVIEW_DIALOG_PAD_PX = 16;
const previewDialogTitle = computed(() => previewDialogAlt.value || _t('Image preview'));

function resolvePreviewBoxSize(): { width: number; height: number } | null {
  const natural = previewDialogNatural.value;
  if (!natural) return null;
  const viewportW = typeof window !== 'undefined' ? window.innerWidth : 1280;
  const viewportH = typeof window !== 'undefined' ? window.innerHeight : 720;
  return resolveImagePreviewDialogSize(natural.width, natural.height, viewportW, viewportH);
}

const previewDialogImgStyle = computed(() => {
  const size = resolvePreviewBoxSize();
  if (!size) {
    return { maxWidth: 'min(92vw, 56rem)', maxHeight: 'min(85vh, 48rem)', width: 'auto', height: 'auto' };
  }
  return { width: `${size.width}px`, height: `${size.height}px` };
});

/** Inline shell size — Dialog portals to body, so parent scoped CSS cannot size it. */
const previewDialogShellStyle = computed(() => {
  const size = resolvePreviewBoxSize();
  const base: Record<string, string> = {
    display: 'block',
    width: 'fit-content',
    height: 'fit-content',
    maxWidth: 'min(92vw, 56rem)',
    maxHeight: 'min(90vh, 56rem)',
    padding: '0.5rem',
    gap: '0',
  };
  if (!size) return base;
  return {
    ...base,
    width: `${size.width + PREVIEW_DIALOG_PAD_PX}px`,
    height: `${size.height + PREVIEW_DIALOG_PAD_PX}px`,
  };
});

function applyPreviewNaturalSize(img: HTMLImageElement | null | undefined): void {
  const width = Number(img?.naturalWidth || 0);
  const height = Number(img?.naturalHeight || 0);
  if (width > 0 && height > 0) {
    previewDialogNatural.value = { width, height };
  }
}

function onPreviewDialogImgLoad(ev: Event): void {
  applyPreviewNaturalSize(ev.target as HTMLImageElement | null);
}

function onThumbLoad(ev: Event): void {
  const img = ev.target as HTMLImageElement | null;
  const width = Number(img?.naturalWidth || 0);
  const height = Number(img?.naturalHeight || 0);
  if (width > 0 && height > 0) {
    thumbNatural.value = { width, height };
  }
}

watch(previewDialogOpen, async open => {
  if (!open) return;
  await nextTick();
  // Cached images may skip @load; read natural size once the img is mounted.
  applyPreviewNaturalSize(previewImgRef.value);
});

const pickErrorRule = {
  validator: (_r: unknown, _value: unknown, cb: (error?: Error) => void) => {
    if (pickError.value) return cb(new Error(pickError.value));
    cb();
  },
} as RuleItem;

const mergedRules = computed<RuleItem[]>(() => [...(props.rules || []), pickErrorRule]);

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

function isRasterImage(file: File): boolean {
  const type = String(file.type || '').toLowerCase();
  if (type && type !== 'application/octet-stream') return RASTER_IMAGE_TYPES.has(type);
  return RASTER_IMAGE_EXT_RE.test(String(file.name || ''));
}

async function validateSelectedImageFile(file: File): Promise<string | undefined> {
  if (!isRasterImage(file)) {
    return _t('Only raster image files are supported.');
  }
  const result = await validateImageFieldFile(file, resolveFieldLimits());
  if (result.ok) return undefined;
  return imageFieldLimitErrorMessage(result);
}

const toView = (raw: any): ViewType => raw;
const fromView = (v: ViewType) => v as unknown as V;
const accept = computed(() => (props.accept || 'image/*').trim() || 'image/*');
const uploadButtonText = computed(() => (props.uploadText || _t('Upload image')).trim() || _t('Upload image'));
const uploadDropLabel = computed(
  () => (props.uploadDropText || _t('Drop image here or click to upload')).trim() || _t('Drop image here or click to upload')
);
const replaceButtonText = computed(() => _t('Replace image'));

function toDisplayText(raw: any): string {
  return toAttachmentDisplayText(raw, '[image]');
}

function previewAlt(raw: unknown): string {
  return resolveFileName(raw) || _t('Image preview');
}

function resolveLinkHref(raw: unknown): string | undefined {
  return resolveDownloadUrl(raw) ?? resolvePreviewUrl(raw);
}

function previewSrc(raw: unknown): string | undefined {
  const url = resolvePreviewUrl(raw);
  if (!url) return undefined;
  if (brokenPreviewUrls.value.has(url)) return undefined;
  const authed = authPreviewBySource.value.get(url);
  if (authed) return authed;
  return url;
}

function markPreviewBroken(url: string): void {
  const next = new Set(brokenPreviewUrls.value);
  next.add(url);
  brokenPreviewUrls.value = next;
}

async function onPreviewError(raw: unknown): Promise<void> {
  const url = resolvePreviewUrl(raw);
  if (!url) return;
  if (authPreviewBySource.value.has(url)) {
    markPreviewBroken(url);
    return;
  }
  if (isInternalDocumentBindingContentUrl(url) && !authPreviewFailed.value.has(url)) {
    const dataUrl = await fetchAuthenticatedPreviewAsDataUrl(url);
    if (dataUrl) {
      const next = new Map(authPreviewBySource.value);
      next.set(url, dataUrl);
      authPreviewBySource.value = next;
      return;
    }
    const failed = new Set(authPreviewFailed.value);
    failed.add(url);
    authPreviewFailed.value = failed;
  }
  markPreviewBroken(url);
}

async function openImagePreview(raw: unknown): Promise<void> {
  const url = resolvePreviewUrl(raw);
  if (!url) return;

  let src = previewSrc(raw);
  if (!src && isInternalDocumentBindingContentUrl(url) && !authPreviewFailed.value.has(url)) {
    const dataUrl = await fetchAuthenticatedPreviewAsDataUrl(url);
    if (dataUrl) {
      const next = new Map(authPreviewBySource.value);
      next.set(url, dataUrl);
      authPreviewBySource.value = next;
      src = dataUrl;
    } else {
      const failed = new Set(authPreviewFailed.value);
      failed.add(url);
      authPreviewFailed.value = failed;
    }
  }
  if (!src) return;

  previewDialogNatural.value = null;
  previewDialogSrc.value = src;
  previewDialogAlt.value = previewAlt(raw);
  previewDialogOpen.value = true;
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

async function applySelectedImage(file: File, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  const err = await validateSelectedImageFile(file);
  if (err) {
    setPickError(err);
    return;
  }
  clearPickError();
  thumbNatural.value = null;
  const valueRef = fieldValue();
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
    // Image fields are embedded in <img>; bind for inline content disposition.
    downloadDisposition: 'inline',
  };

  if (typeof onFieldChange === 'function') {
    await onFieldChange();
  }
}

async function onPickedFile(file: File, fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  await applySelectedImage(file, fieldValue, onFieldChange);
}

async function removeImage(fieldValue: ValueRefGetter, onFieldChange?: OnFieldChange): Promise<void> {
  clearPickError();
  thumbNatural.value = null;
  const valueRef = fieldValue();
  valueRef.value = null;
  if (typeof onFieldChange === 'function') {
    await onFieldChange();
  }
}
</script>

<!-- Unscoped: DialogContent teleports to body; parent scoped/:deep selectors never match. -->
<style>
.choy-image-preview-dialog.choy-image-preview-dialog {
  display: block !important;
  width: fit-content !important;
  height: fit-content !important;
  max-width: none !important;
  max-height: none !important;
  gap: 0 !important;
  padding: 0 !important;
  border-width: 0 !important;
  background: transparent !important;
  box-shadow: none !important;
}

.choy-image-preview-dialog__shell {
  box-sizing: border-box;
}

.choy-image-preview-dialog__img {
  display: block;
  vertical-align: top;
  object-fit: contain;
}

/* Keep the kit close control visible over the transparent outer panel. */
.choy-image-preview-dialog > button {
  z-index: 2;
}
</style>
