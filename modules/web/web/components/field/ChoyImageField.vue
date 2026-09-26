<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    data-anchor="choy.image-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="error || fileError"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, labelId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <div class="flex flex-col gap-2">
        <input
          ref="inputRef"
          type="file"
          :accept="RASTER_IMAGE_ACCEPT"
          class="hidden"
          :id="controlId"
          :disabled="disabled || readonly"
          tabindex="-1"
          aria-hidden="true"
          @change="onChange"
        />
        <div class="flex flex-wrap items-center gap-2">
          <ChoyButton
            type="button"
            variant="outline"
            size="sm"
            :disabled="disabled || readonly"
            :aria-labelledby="label ? labelId : undefined"
            :aria-invalid="ariaInvalid"
            :aria-required="ariaRequired"
            :aria-describedby="ariaDescribedby"
            @click="onPick"
          >
            Choose image
          </ChoyButton>
          <span v-if="model?.name" class="text-sm text-foreground">{{ model.name }}</span>
          <ChoyButton
            v-if="model"
            type="button"
            variant="ghost"
            size="sm"
            :disabled="disabled || readonly"
            @click="onClear"
          >
            Clear
          </ChoyButton>
        </div>
        <img
          v-if="previewSrc"
          :src="previewSrc"
          :alt="model?.name || 'Image preview'"
          class="max-h-40 max-w-full rounded-md border border-border object-contain"
        />
      </div>
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import type { ClassValue } from '../../lib/utils';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/** Image field value with optional preview URL (object URL or data URL). */
export type ChoyImageValue = {
  name: string;
  size: number;
  previewUrl?: string;
  file?: File;
} | null;

/**
 * Image field: raster image pick with preview from object URL or provided previewUrl.
 * SVG is rejected (and omitted from `accept`) because hosts may later serve it
 * in a script-capable context.
 */

/** Single source of truth for the picker `accept` attr and pick validation. */
const RASTER_IMAGE_TYPES = [
  'image/png',
  'image/jpeg',
  'image/gif',
  'image/webp',
  'image/bmp',
  'image/avif',
] as const;
const RASTER_IMAGE_ACCEPT = RASTER_IMAGE_TYPES.join(',');
const RASTER_IMAGE_TYPE_SET: ReadonlySet<string> = new Set(RASTER_IMAGE_TYPES);
const RASTER_IMAGE_EXT_RE = new RegExp(
  `\\.(${RASTER_IMAGE_TYPES.map((t) => {
    const subtype = t.slice('image/'.length);
    return subtype === 'jpeg' ? 'jpe?g' : subtype;
  }).join('|')})$`,
  'i',
);

const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
    }
  >(),
  { ...choyFieldChromeDefaults },
);

const model = defineModel<ChoyImageValue>({ default: null });
const inputRef = ref<HTMLInputElement | null>(null);
const localObjectUrl = ref<string | null>(null);
/** Set when a picked file is rejected; the model stays untouched. */
const fileError = ref('');

const previewSrc = computed(
  () => localObjectUrl.value || model.value?.previewUrl || '',
);

watch(
  // Only the picked File owns the local object URL; a host previewUrl change must not
  // revoke/re-create it (that would reload the blob preview needlessly).
  () => model.value?.file,
  (file) => {
    if (localObjectUrl.value) {
      URL.revokeObjectURL(localObjectUrl.value);
      localObjectUrl.value = null;
    }
    if (file) {
      localObjectUrl.value = URL.createObjectURL(file);
    }
  },
  { immediate: true },
);

watch(
  // A host-driven value change (e.g. loading another record) must drop a stale
  // pick-rejection error; a rejected pick never changes the model, so it survives.
  () => model.value,
  () => {
    fileError.value = '';
  },
);

onBeforeUnmount(() => {
  if (localObjectUrl.value) {
    URL.revokeObjectURL(localObjectUrl.value);
    localObjectUrl.value = null;
  }
});

function onPick(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  inputRef.value?.click();
}

function onChange(event: Event): void {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) {
    return;
  }
  // `accept` is only a hint; validate against the same raster allow-list the
  // input offers (SVG stays excluded: hosts may serve it in a script-capable context).
  const reportedType = file.type.toLowerCase();
  // Some OS/browser pairs report `application/octet-stream` for valid images;
  // fall back to the extension instead of rejecting the pick outright.
  const isImage =
    reportedType !== '' && reportedType !== 'application/octet-stream'
      ? RASTER_IMAGE_TYPE_SET.has(reportedType)
      : RASTER_IMAGE_EXT_RE.test(file.name);
  if (!isImage) {
    fileError.value = 'Only raster image files are supported.';
    input.value = '';
    return;
  }
  // Let the model watcher own create/revoke of the local object URL.
  fileError.value = '';
  model.value = { name: file.name, size: file.size, file };
  input.value = '';
}

function onClear(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  fileError.value = '';
  model.value = null;
}
</script>
