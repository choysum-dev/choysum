<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
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
/** Set when a picked file is rejected by `accept`; the model stays untouched. */
const fileError = ref('');

watch(
  // A host-driven value change (e.g. loading another record) must drop a stale
  // pick-rejection error; a rejected pick never changes the model, so it survives.
  () => model.value,
  () => {
    fileError.value = '';
  },
);

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
  // `accept` is only a hint; drop files the host did not ask for.
  const accept = props.accept.trim().toLowerCase();
  if (accept) {
    const name = file.name.toLowerCase();
    const type = file.type.toLowerCase();
    const allowed = accept
      .split(',')
      .map((token) => token.trim())
      .filter(Boolean)
      .some((token) => {
        // `*` / `*/*` mean "any type"; they must not reject every file.
        if (token === '*' || token === '*/*') {
          return true;
        }
        if (token.startsWith('.')) {
          return name.endsWith(token);
        }
        // Authors commonly write `*.pdf`; browsers ignore such tokens, so match
        // by extension instead of rejecting every pick.
        if (token.startsWith('*.')) {
          return name.endsWith(token.slice(1));
        }
        if (token.endsWith('/*')) {
          // An empty `file.type` cannot be matched against a wildcard; don't
          // reject a pick we cannot verify here — the host still validates uploads.
          return type === '' || type.startsWith(token.slice(0, -1));
        }
        // Some OS/browser pairs report an empty `file.type`; fall back to the
        // extension so a valid pick is not rejected outright.
        const subtype = token.split('/')[1] ?? '';
        // `image/jpeg` → `.jpg`/`.jpeg`; `image/svg+xml` → `.svg` (not `.svg+xml`).
        const fallbackExts =
          subtype === 'jpeg'
            ? ['.jpg', '.jpeg']
            : token === 'image/svg+xml'
              ? ['.svg']
              : subtype
                ? [`.${subtype}`]
                : [];
        return (
          type === token ||
          (type === '' && fallbackExts.some((ext) => name.endsWith(ext)))
        );
      });
    if (!allowed) {
      fileError.value = 'Selected file type is not allowed.';
      input.value = '';
      return;
    }
  }
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

<template>
  <ChoyFieldBase
    data-anchor="choy.binary-field"
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
