<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref } from 'vue';
import type { ClassValue } from '../../lib/utils';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/** Binary attachment value: metadata plus optional File for upload. */
export type ChoyBinaryValue = {
  name: string;
  size: number;
  file?: File;
} | null;

/**
 * Binary file field: file input plus filename/size display.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      accept?: string;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    accept: '',
  },
);

const model = defineModel<ChoyBinaryValue>({ default: null });
const inputRef = ref<HTMLInputElement | null>(null);

const displayName = computed(() => model.value?.name ?? '');
const displaySize = computed(() => {
  const size = model.value?.size;
  if (typeof size !== 'number' || !Number.isFinite(size) || size < 0) {
    return '';
  }
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
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
    data-anchor="choy.binary-field"
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
    <template #default="{ controlId, labelId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <div class="flex flex-wrap items-center gap-2">
        <input
          ref="inputRef"
          type="file"
          class="hidden"
          :id="controlId"
          :accept="accept || undefined"
          :disabled="disabled || readonly"
          tabindex="-1"
          aria-hidden="true"
          @change="onChange"
        />
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
          Choose file
        </ChoyButton>
        <span v-if="displayName" class="text-sm text-foreground">
          {{ displayName }}
          <span v-if="displaySize" class="text-foreground/60">({{ displaySize }})</span>
        </span>
        <span v-else class="text-sm text-foreground/50">No file chosen</span>
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
    </template>
  </ChoyFieldBase>
</template>
