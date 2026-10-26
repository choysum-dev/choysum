<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    :class="[
      'choy-attachment-field flex w-full flex-col items-stretch gap-3',
      rootClass,
    ]"
  >
    <!-- Image form presentation (Odoo-style avatar when valued); empty edit uses compact upload. -->
    <template v-if="useImagePresentation">
      <template v-if="mode === 'edit'">
        <div
          v-if="hasValue"
          class="choy-image-avatar group/image-avatar relative overflow-hidden border border-border bg-muted"
          :class="[{ 'opacity-60': actionsDisabled }, avatarRadiusClass, currentClass]"
          :style="avatarBoxStyle"
          :data-fit="effectiveFit"
          :data-state="attachmentState"
        >
          <a
            v-if="openHref"
            class="choy-image-avatar__hit absolute inset-0 z-0 block cursor-pointer"
            :href="actionsDisabled ? undefined : openHref"
            :download="downloadFileName || true"
            :aria-disabled="actionsDisabled || undefined"
            :aria-label="openLabel"
            rel="noopener noreferrer"
            @click="onDownloadClick"
          />
          <div class="pointer-events-none absolute inset-0 z-[1] flex items-center justify-center overflow-hidden">
            <slot name="media" />
          </div>
          <TooltipProvider :delay-duration="300">
            <div
              class="choy-image-avatar__overlay pointer-events-none absolute inset-x-0 bottom-0 z-[2] flex items-center justify-center gap-0.5 bg-black/55 px-1 py-1 opacity-0 transition-opacity group-hover/image-avatar:pointer-events-auto group-hover/image-avatar:opacity-100 group-focus-within/image-avatar:pointer-events-auto group-focus-within/image-avatar:opacity-100"
            >
              <Tooltip v-if="openHref">
                <TooltipTrigger as-child>
                  <a
                    class="inline-flex size-7 items-center justify-center rounded text-white hover:bg-white/15"
                    :class="{ 'pointer-events-none opacity-50': actionsDisabled }"
                    :href="actionsDisabled ? undefined : openHref"
                    :download="downloadFileName || true"
                    :aria-disabled="actionsDisabled || undefined"
                    :aria-label="openLabel"
                    rel="noopener noreferrer"
                    @click.stop="onDownloadClick"
                  >
                    <Download class="size-3.5" aria-hidden="true" />
                  </a>
                </TooltipTrigger>
                <TooltipContent side="top">{{ openLabel }}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger as-child>
                  <label class="inline-flex" @click.stop>
                    <input
                      type="file"
                      class="sr-only"
                      :id="inputId || undefined"
                      :name="inputName || undefined"
            :aria-describedby="describedBy || undefined"
            :aria-invalid="invalid ? 'true' : undefined"
                      :accept="acceptAttr"
                      :disabled="actionsDisabled"
                      @change="onFileInputChange"
                    />
                    <ChoyButton
                      size="sm"
                      variant="ghost"
                      class="size-7 p-0 text-white hover:bg-white/15 hover:text-white"
                      :disabled="actionsDisabled"
                      :aria-label="replaceLabel"
                      as="span"
                    >
                      <RefreshCw class="size-3.5" aria-hidden="true" />
                    </ChoyButton>
                  </label>
                </TooltipTrigger>
                <TooltipContent side="top">{{ replaceLabel }}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger as-child>
                  <ChoyButton
                    size="sm"
                    variant="ghost"
                    class="size-7 p-0 text-white hover:bg-white/15 hover:text-white"
                    :disabled="actionsDisabled"
                    :aria-label="removeLabel"
                    @click.stop="emit('remove')"
                  >
                    <Trash2 class="size-3.5" aria-hidden="true" />
                  </ChoyButton>
                </TooltipTrigger>
                <TooltipContent side="top">{{ removeLabel }}</TooltipContent>
              </Tooltip>
            </div>
          </TooltipProvider>
        </div>
        <label
          v-else
          :class="['choy-attachment-upload box-border block w-full cursor-pointer', uploadClass]"
          @dragover.prevent="onDragOver"
          @drop.prevent="onDrop"
        >
          <input
            type="file"
            class="sr-only"
            :id="inputId || undefined"
            :name="inputName || undefined"
            :aria-describedby="describedBy || undefined"
            :aria-invalid="invalid ? 'true' : undefined"
            :accept="acceptAttr"
            :disabled="actionsDisabled"
            @change="onFileInputChange"
          />
          <Attachment
            size="xs"
            :state="attachmentState === 'uploading' ? 'uploading' : 'idle'"
            class="choy-attachment-upload--compact w-full max-w-full min-h-control flex-nowrap items-center rounded-md border border-dashed border-input bg-transparent px-3 hover:border-primary hover:bg-primary-subtle"
            :class="{ 'opacity-60': actionsDisabled }"
          >
            <Upload class="size-4 shrink-0 text-primary" />
            <div class="min-w-0 flex-1 truncate text-sm leading-none text-muted-foreground">
              {{ drag ? dropLabel : uploadLabel }}
            </div>
          </Attachment>
        </label>
      </template>
      <template v-else>
        <button
          v-if="hasValue && imageClickAction === 'preview'"
          type="button"
          class="choy-image-avatar choy-image-avatar--display relative block overflow-hidden border border-border bg-muted p-0 text-inherit"
          :class="avatarRadiusClass"
          :style="avatarBoxStyle"
          :data-fit="effectiveFit"
          :aria-label="previewLabel || openLabel"
          @click.stop="onPreviewClick"
        >
          <div class="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden">
            <slot name="media" />
          </div>
        </button>
        <a
          v-else-if="hasValue && openHref"
          class="choy-image-avatar choy-image-avatar--display relative block overflow-hidden border border-border bg-muted text-inherit no-underline"
          :class="avatarRadiusClass"
          :style="avatarBoxStyle"
          :data-fit="effectiveFit"
          :href="openHref"
          :download="downloadFileName || true"
          :aria-label="openLabel"
          rel="noopener noreferrer"
          @click.stop="onDownloadClick"
        >
          <div class="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden">
            <slot name="media" />
          </div>
        </a>
        <div
          v-else-if="hasValue"
          class="choy-image-avatar choy-image-avatar--display relative overflow-hidden border border-border bg-muted"
          :class="avatarRadiusClass"
          :style="avatarBoxStyle"
          :data-fit="effectiveFit"
        >
          <div class="absolute inset-0 flex items-center justify-center overflow-hidden">
            <slot name="media" />
          </div>
        </div>
        <span v-else class="choy-field-display-empty inline-flex min-h-control items-center text-muted-foreground" />
      </template>
    </template>

    <!-- File presentation (Binary + Image table cells). -->
    <template v-else>
      <template v-if="mode === 'edit'">
        <Attachment
          v-if="hasValue"
          :size="compact ? 'xs' : 'default'"
          :class="[
            'choy-attachment-current max-w-full',
            compact
              ? 'choy-attachment-current--compact w-full min-h-control flex-nowrap items-center rounded-md border-input bg-transparent px-3 shadow-xs'
              : 'w-full bg-muted',
            currentClass,
          ]"
          :state="attachmentState"
          :title="tooltip || undefined"
        >
          <slot name="media" />
          <AttachmentContent :class="compact ? 'min-w-0 flex-1' : undefined">
            <AttachmentTitle :class="compact ? 'truncate' : undefined">{{ title }}</AttachmentTitle>
            <AttachmentDescription v-if="meta" :class="compact ? 'truncate' : undefined">{{ meta }}</AttachmentDescription>
            <AttachmentActions :class="compact ? 'shrink-0' : undefined">
              <a
                v-if="openHref"
                class="inline-flex h-auto items-center px-1.5 py-0 text-sm text-primary hover:underline"
                :class="{ 'pointer-events-none opacity-50': actionsDisabled }"
                :href="actionsDisabled ? undefined : openHref"
                :download="downloadFileName || true"
                :aria-disabled="actionsDisabled || undefined"
                rel="noopener noreferrer"
                @click.stop="onDownloadClick"
              >
                {{ openLabel }}
              </a>
              <label class="inline-flex">
                <input
                  type="file"
                  class="sr-only"
                  :id="inputId || undefined"
                  :name="inputName || undefined"
            :aria-describedby="describedBy || undefined"
            :aria-invalid="invalid ? 'true' : undefined"
                  :accept="acceptAttr"
                  :disabled="actionsDisabled"
                  @change="onFileInputChange"
                />
                <ChoyButton size="sm" variant="ghost" class="h-auto px-1.5 py-0 text-primary" :disabled="actionsDisabled" as="span">
                  {{ replaceLabel }}
                </ChoyButton>
              </label>
              <ChoyButton
                size="sm"
                variant="ghost"
                class="h-auto px-1.5 py-0 text-destructive hover:text-destructive"
                :disabled="actionsDisabled"
                @click="emit('remove')"
              >
                {{ removeLabel }}
              </ChoyButton>
            </AttachmentActions>
          </AttachmentContent>
        </Attachment>
        <label
          v-else
          :class="['choy-attachment-upload box-border block w-full cursor-pointer', uploadClass]"
          @dragover.prevent="onDragOver"
          @drop.prevent="onDrop"
        >
          <input
            type="file"
            class="sr-only"
            :id="inputId || undefined"
            :name="inputName || undefined"
            :aria-describedby="describedBy || undefined"
            :aria-invalid="invalid ? 'true' : undefined"
            :accept="acceptAttr"
            :disabled="actionsDisabled"
            @change="onFileInputChange"
          />
          <Attachment
            :size="compact ? 'xs' : 'default'"
            :state="attachmentState === 'uploading' ? 'uploading' : 'idle'"
            class="w-full max-w-full"
            :class="
              compact
                ? 'choy-attachment-upload--compact min-h-control flex-nowrap items-center rounded-md border border-dashed border-input bg-transparent px-3 hover:border-primary hover:bg-primary-subtle'
                : { 'choy-attachment-upload--drag min-h-32 justify-center px-4 py-5 border-dashed hover:border-primary hover:bg-primary-subtle': drag }
            "
          >
            <template v-if="compact">
              <Upload class="size-4 shrink-0 text-primary" />
              <div class="min-w-0 flex-1 truncate text-sm leading-none text-muted-foreground">
                {{ drag ? dropLabel : uploadLabel }}
              </div>
            </template>
            <template v-else-if="drag">
              <Upload class="mx-auto mb-2.5 block size-5 text-primary" />
              <div class="w-full text-center text-sm leading-normal text-muted-foreground">{{ dropLabel }}</div>
            </template>
            <ChoyButton v-else size="sm" variant="ghost" class="h-auto px-1.5 py-0 text-primary" as="span">{{ uploadLabel }}</ChoyButton>
          </Attachment>
        </label>
      </template>

      <template v-else>
        <Attachment
          v-if="hasValue && table"
          size="sm"
          :class="['border-transparent bg-transparent shadow-none', tableClass]"
          state="done"
        >
          <slot name="media" />
          <AttachmentContent>
            <AttachmentTitle>{{ title }}</AttachmentTitle>
          </AttachmentContent>
        </Attachment>
        <a
          v-else-if="hasValue && openHref"
          class="choy-attachment-display-card inline-flex max-w-full text-inherit no-underline"
          :href="openHref"
          :download="downloadFileName || true"
          :title="tooltip || title"
          :aria-label="openLabel"
          rel="noopener noreferrer"
          @click.stop="onDownloadClick"
        >
          <Attachment
            :size="compact ? 'xs' : 'default'"
            :class="
              compact
                ? 'choy-attachment-display-card--compact max-w-full min-h-control cursor-pointer flex-nowrap items-center rounded-md border-transparent bg-transparent px-1.5 shadow-none transition-colors hover:bg-muted/50'
                : 'choy-attachment-display-card--interactive max-w-full cursor-pointer bg-muted transition-colors hover:border-primary-soft hover:bg-primary-subtle'
            "
            state="done"
          >
            <slot name="media" />
            <AttachmentContent :class="compact ? 'min-w-0 flex-1' : undefined">
              <AttachmentTitle :class="compact ? 'truncate' : undefined">{{ title }}</AttachmentTitle>
              <AttachmentDescription v-if="meta" :class="compact ? 'truncate text-muted-foreground' : undefined">
                {{ meta }}
              </AttachmentDescription>
            </AttachmentContent>
            <AttachmentActions v-if="compact" class="shrink-0 text-muted-foreground">
              <Download class="size-3.5" aria-hidden="true" />
            </AttachmentActions>
          </Attachment>
        </a>
        <Attachment
          v-else-if="hasValue"
          :size="compact ? 'xs' : 'default'"
          :class="
            compact
              ? 'choy-attachment-display-card--compact max-w-full min-h-control flex-nowrap items-center rounded-md border-transparent bg-transparent px-1.5 shadow-none'
              : 'choy-attachment-display-card max-w-full bg-muted'
          "
          state="done"
        >
          <slot name="media" />
          <AttachmentContent :class="compact ? 'min-w-0 flex-1' : undefined">
            <AttachmentTitle :class="compact ? 'truncate' : undefined">{{ title }}</AttachmentTitle>
            <AttachmentDescription v-if="meta" :class="compact ? 'truncate' : undefined">{{ meta }}</AttachmentDescription>
          </AttachmentContent>
        </Attachment>
        <span v-else class="choy-field-display-empty inline-flex min-h-control items-center text-muted-foreground" />
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Download, RefreshCw, Trash2, Upload } from 'lucide-vue-next';
import ChoyButton from '@/web/web/components/layout/ChoyButton.vue';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../vendor/ui/tooltip';
import Attachment from '../vendor/ui/attachment/Attachment.vue';
import AttachmentActions from '../vendor/ui/attachment/AttachmentActions.vue';
import AttachmentContent from '../vendor/ui/attachment/AttachmentContent.vue';
import AttachmentDescription from '../vendor/ui/attachment/AttachmentDescription.vue';
import AttachmentTitle from '../vendor/ui/attachment/AttachmentTitle.vue';
import { downloadAttachmentFromUrl } from './attachmentPreviewAuth';
import {
  resolveImageAvatarBoxStyle,
  resolveImageAvatarContainBoxStyle,
  resolveImageAvatarRadiusClass,
  type ImageAvatarFit,
  type ImageAvatarRadius,
} from './attachmentFieldHelpers';

defineOptions({ name: 'ChoyAttachmentFieldChrome' });

const props = withDefaults(
  defineProps<{
    mode: 'edit' | 'display';
    hasValue: boolean;
    title: string;
    meta?: string;
    openHref?: string;
    /** Suggested filename for the browser download attribute / save dialog. */
    downloadFileName?: string;
    /** Form image avatar vs file chip. Table cells always use file chrome. */
    presentation?: 'file' | 'image';
    /** Avatar box width (px number or CSS length). Default 112. For contain, acts as max width. */
    width?: number | string;
    /** Avatar box height (px number or CSS length). For contain, acts as max height. */
    height?: number | string;
    /** cover crops to the box; contain preserves natural ratio within max width/height. */
    fit?: ImageAvatarFit;
    /** Corner radius token for the form image box. */
    radius?: ImageAvatarRadius;
    /** Intrinsic pixels used when fit=contain. */
    naturalWidth?: number;
    naturalHeight?: number;
    /**
     * Image form primary click: download bytes, or emit preview for an in-page viewer.
     * Edit overlays keep an explicit Download action when openHref is set.
     */
    imageClickAction?: 'download' | 'preview';
    /** Accessible label when imageClickAction is preview. */
    previewLabel?: string;
    /**
     * Compact file row: single-line height, truncated name, optional borderless display.
     * Used by Binary (and similar file chips).
     */
    compact?: boolean;
    /** Native title/tooltip for compact rows (e.g. full mime · size). */
    tooltip?: string;
    /** Labelable file input id (matches FieldLabel `for`). */
    inputId?: string;
    /** Native name for autofill / form association. */
    inputName?: string;
    /** Field help/error ids for the hidden file input. */
    describedBy?: string;
    /** When true, marks the file input aria-invalid. */
    invalid?: boolean;
    accept?: string;
    drag?: boolean;
    disabled?: boolean;
    /** Form submit / upload in flight. */
    busy?: boolean;
    attachmentState?: 'idle' | 'uploading' | 'processing' | 'error' | 'done';
    uploadLabel: string;
    dropLabel: string;
    replaceLabel: string;
    removeLabel: string;
    openLabel: string;
    table?: boolean;
    rootClass?: string;
    currentClass?: string;
    uploadClass?: string;
    tableClass?: string;
  }>(),
  {
    meta: undefined,
    openHref: undefined,
    downloadFileName: undefined,
    presentation: 'file',
    width: undefined,
    height: undefined,
    fit: 'contain',
    radius: 'md',
    naturalWidth: undefined,
    naturalHeight: undefined,
    imageClickAction: 'download',
    previewLabel: undefined,
    compact: false,
    tooltip: undefined,
    inputId: undefined,
    inputName: undefined,
    describedBy: undefined,
    invalid: false,
    accept: '',
    drag: true,
    disabled: false,
    busy: false,
    attachmentState: undefined,
    table: false,
    rootClass: '',
    currentClass: '',
    uploadClass: '',
    tableClass: 'max-w-36',
  }
);

const emit = defineEmits<{
  file: [file: File];
  remove: [];
  preview: [];
}>();

const acceptAttr = computed(() => {
  const a = (props.accept || '').trim();
  return a || undefined;
});

const actionsDisabled = computed(() => props.disabled || props.busy);

const useImagePresentation = computed(() => props.presentation === 'image' && !props.table);

const effectiveFit = computed<ImageAvatarFit>(() => (props.fit === 'cover' ? 'cover' : 'contain'));

const avatarRadiusClass = computed(() => resolveImageAvatarRadiusClass(props.radius));

const avatarBoxStyle = computed(() => {
  if (effectiveFit.value === 'cover') {
    return resolveImageAvatarBoxStyle(props.width, props.height);
  }
  return resolveImageAvatarContainBoxStyle({
    width: props.width,
    height: props.height,
    naturalWidth: props.naturalWidth,
    naturalHeight: props.naturalHeight,
  });
});

const attachmentState = computed(() => {
  if (props.attachmentState) return props.attachmentState;
  if (props.busy && props.hasValue) return 'uploading';
  if (props.hasValue) return 'done';
  return 'idle';
});

function onFileInputChange(ev: Event): void {
  if (actionsDisabled.value) return;
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  input.value = '';
  emit('file', file);
}

async function onDownloadClick(ev: MouseEvent): Promise<void> {
  if (actionsDisabled.value) {
    ev.preventDefault();
    return;
  }
  const href = (props.openHref || '').trim();
  if (!href) {
    ev.preventDefault();
    return;
  }
  ev.preventDefault();
  const fileName = props.downloadFileName || props.title;
  const ok = await downloadAttachmentFromUrl(href, fileName);
  if (!ok && typeof window !== 'undefined') {
    // Last resort when auth/fetch fails: navigate so the click is not a no-op.
    window.open(href, '_blank', 'noopener,noreferrer');
  }
}

function onPreviewClick(): void {
  if (actionsDisabled.value) return;
  emit('preview');
}

function onDragOver(ev: DragEvent): void {
  if (actionsDisabled.value) return;
  if (ev.dataTransfer) ev.dataTransfer.dropEffect = 'copy';
}

function onDrop(ev: DragEvent): void {
  if (actionsDisabled.value) return;
  const file = ev.dataTransfer?.files?.[0];
  if (!file) return;
  emit('file', file);
}
</script>

<style scoped>
.choy-image-avatar :deep([data-slot='attachment-media']),
.choy-image-avatar-upload :deep([data-slot='attachment-media']) {
  width: 100%;
  height: 100%;
  aspect-ratio: auto;
  border-radius: 0;
  border-width: 0;
  background: transparent;
  /* Kit image media defaults to opacity-60 outside Attachment[state=done|idle]. */
  opacity: 1;
}

.choy-image-avatar :deep(img),
.choy-image-avatar-upload :deep(img) {
  width: 100%;
  height: 100%;
}

.choy-image-avatar[data-fit='contain'] :deep(img) {
  object-fit: contain;
}

.choy-image-avatar[data-fit='cover'] :deep(img),
.choy-image-avatar:not([data-fit]) :deep(img) {
  object-fit: cover;
}

.choy-attachment-current--compact,
.choy-attachment-upload--compact {
  box-sizing: border-box;
  min-height: var(--choy-control-height, 2rem);
  padding-block: 0 !important;
  padding-inline: 0.75rem !important;
}

.choy-attachment-display-card--compact {
  box-sizing: border-box;
  min-height: var(--choy-control-height, 2rem);
  padding-block: 0 !important;
  padding-inline: 0.375rem !important;
}

.choy-attachment-current--compact :deep([data-slot='attachment-media']),
.choy-attachment-display-card--compact :deep([data-slot='attachment-media']) {
  width: auto;
  aspect-ratio: auto;
  background-color: transparent;
  border-radius: 0;
}

.choy-attachment-current--compact :deep([data-slot='attachment-content']),
.choy-attachment-display-card--compact :deep([data-slot='attachment-content']) {
  display: flex;
  min-width: 0;
  flex: 1 1 auto;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  gap: 0.5rem;
  padding: 0 !important;
}

.choy-attachment-current--compact :deep([data-slot='attachment-title']),
.choy-attachment-display-card--compact :deep([data-slot='attachment-title']) {
  flex: 1 1 auto;
  min-width: 0;
  line-height: 1.25;
}

.choy-attachment-current--compact :deep([data-slot='attachment-description']),
.choy-attachment-display-card--compact :deep([data-slot='attachment-description']) {
  margin-top: 0 !important;
  flex: 0 0 auto;
  max-width: 6.5rem;
  line-height: 1.25;
}

.choy-attachment-current--compact :deep([data-slot='attachment-actions']),
.choy-attachment-display-card--compact :deep([data-slot='attachment-actions']) {
  margin-inline-start: auto;
}

.choy-attachment-upload--compact :deep([data-slot='attachment-media']) {
  display: none;
}
</style>
