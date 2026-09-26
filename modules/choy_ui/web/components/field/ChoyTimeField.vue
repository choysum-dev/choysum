<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

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
 * Time field via native time input. Model is `HH:mm` (or `HH:mm:ss`) or null.
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

/** Native `time` only accepts `HH:mm[:ss]`. */
const displayValue = computed(() => {
  const normalized = String(model.value ?? '').trim();
  const match = /^(\d{1,2}):(\d{2})(:\d{2})?/.exec(normalized);
  // Native `time` inputs require a two-digit hour; a model like "9:30" would
  // otherwise render as an empty control and hide the loaded value.
  return match ? `${match[1]!.padStart(2, '0')}:${match[2]}${match[3] ?? ''}` : '';
});

/** Truncates text to the minute precision a native `time` input often reports. */
function toInputPrecision(text: string): string {
  const match = String(text ?? '')
    .trim()
    .match(/^(\d{1,2}):(\d{2})/);
  return match ? `${match[1]!.padStart(2, '0')}:${match[2]}` : '';
}

function onInput(value: string): void {
  // Browsers may ignore `readonly` on native time inputs, so guard the update too.
  if (props.readonly || props.disabled) {
    return;
  }
  // A re-pick of the current time must not rewrite the host model and strip an
  // offset/second-bearing value (native inputs often omit seconds).
  if (value !== '' && toInputPrecision(value) === toInputPrecision(displayValue.value)) {
    return;
  }
  model.value = value === '' ? null : value;
}
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.time-field"
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
        type="time"
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
