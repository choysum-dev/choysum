<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import Input from '../vendor/ui/input/Input.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  formatChoyMonetary,
  parseChoyNumber,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Monetary amount field. Focus shows raw number; blur shows formatted text.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      precision?: number;
      currency?: string;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: '',
    precision: 2,
    currency: '',
  },
);

const model = defineModel<number | null>({ default: null });
const focused = ref(false);
const draft = ref('');

const displayValue = computed(() => {
  if (focused.value) {
    return draft.value;
  }
  return formatChoyMonetary(model.value, {
    precision: props.precision,
    currency: props.currency,
  });
});

watch(model, (next) => {
  if (!focused.value) {
    draft.value = next === null || next === undefined ? '' : String(next);
  }
});

function onFocus(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  focused.value = true;
  draft.value = model.value === null || model.value === undefined ? '' : String(model.value);
}

function onBlur(): void {
  focused.value = false;
  if (props.readonly || props.disabled) {
    return;
  }
  const parsed = parseChoyNumber(draft.value, 'decimal');
  model.value = parsed;
  draft.value = parsed === null ? '' : String(parsed);
}

function onInput(value: string): void {
  draft.value = value;
}
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.monetary-field"
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
    <Input
      :id="name || undefined"
      :model-value="displayValue"
      :name="name || undefined"
      :placeholder="placeholder"
      :disabled="disabled || readonly"
      :readonly="readonly"
      inputmode="decimal"
      @update:model-value="onInput"
      @focus="onFocus"
      @blur="onBlur"
    />
  </ChoyFieldBase>
</template>
