<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

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
 * Image field: accept=image/* with preview from object URL or provided previewUrl.
 */
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

const previewSrc = computed(
  () => model.value?.previewUrl || localObjectUrl.value || '',
);

watch(
  model,
  (next) => {
    if (localObjectUrl.value) {
      URL.revokeObjectURL(localObjectUrl.value);
      localObjectUrl.value = null;
    }
    if (next?.file && !next.previewUrl) {
      localObjectUrl.value = URL.createObjectURL(next.file);
    }
  },
  { immediate: true },
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
  // Let the model watcher own create/revoke of the local object URL.
  model.value = { name: file.name, size: file.size, file };
  input.value = '';
}

function onClear(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  model.value = null;
}
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.image-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="error"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, ariaInvalid, ariaDescribedby }">
      <div class="flex flex-col gap-2">
        <input
          ref="inputRef"
          type="file"
          accept="image/*"
          class="hidden"
          :disabled="disabled || readonly"
          @change="onChange"
        />
        <div class="flex flex-wrap items-center gap-2">
          <ChoyButton
            type="button"
            variant="outline"
            size="sm"
            :id="controlId"
            :disabled="disabled || readonly"
            :aria-invalid="ariaInvalid"
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
