<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <ChoyFieldBase
    data-anchor="choy.datetime-field"
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
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <Input
        :id="controlId"
        type="datetime-local"
        :model-value="displayValue"
        :name="name || undefined"
        :disabled="disabled"
        :readonly="readonly"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
        @update:model-value="onInput"
      />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import Input from '../vendor/ui/input/Input.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Datetime field via native datetime-local input (DatePicker is date-only).
 * Model is a string (typically `YYYY-MM-DDTHH:mm`) or null.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
    }
  >(),
  { ...choyFieldChromeDefaults },
);

const model = defineModel<string | null>({ default: null });

/** Native `datetime-local` only accepts `YYYY-MM-DDTHH:mm[:ss]`. */
const displayValue = computed(() => {
  const normalized = String(model.value ?? '')
    .trim()
    .replace(' ', 'T');
  return normalized.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2})?/)?.[0] ?? '';
});

/** Truncates text to the minute precision a native `datetime-local` often reports. */
function toInputPrecision(text: string): string {
  return String(text ?? '')
    .trim()
    .replace(' ', 'T')
    .slice(0, 16);
}

function onInput(value: string): void {
  // Browsers may ignore `readonly` on native datetime-local inputs, so guard the update too.
  if (props.readonly || props.disabled) {
    return;
  }
  // Don't echo a no-op pick back into the host model: a value like
  // `2026-09-23T10:30:45Z` must not become `...T10:30` when the user re-picks
  // the same minute (native inputs often omit seconds).
  if (value !== '' && toInputPrecision(value) === toInputPrecision(displayValue.value)) {
    return;
  }
  model.value = value === '' ? null : value;
}
</script>
