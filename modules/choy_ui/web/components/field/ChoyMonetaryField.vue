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
  resolveChoyMonetaryPrecision,
  roundChoyDecimal,
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
/** True after the user types; keeps draft visible across blur when still invalid. */
const edited = ref(false);
/** True when blur kept an unparsed draft while the host model is unchanged. */
const invalidDraft = ref(false);
const draft = ref('');

const displayValue = computed(() => {
  if (focused.value || edited.value) {
    return draft.value;
  }
  return formatChoyMonetary(model.value, {
    precision: props.precision,
    currency: props.currency,
  });
});

watch(model, (next) => {
  // Preserve an in-progress edit only when the host value already matches the draft.
  const parsedDraft = parseChoyNumber(draft.value, 'decimal');
  if (
    focused.value &&
    edited.value &&
    parsedDraft !== null &&
    next !== null &&
    parsedDraft === next
  ) {
    return;
  }
  // Host-driven updates (e.g. loading another record) must win over a stale invalid draft.
  edited.value = false;
  invalidDraft.value = false;
  draft.value = next === null || next === undefined ? '' : String(next);
});

function onFocus(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  focused.value = true;
  invalidDraft.value = false;
  if (!edited.value) {
    draft.value =
      model.value === null || model.value === undefined ? '' : String(model.value);
  }
}

function onBlur(): void {
  focused.value = false;
  if (props.readonly || props.disabled) {
    return;
  }
  if (!edited.value) {
    // Focus/blur alone must not rewrite the host model: re-committing the raw text
    // would silently round it (12.345 -> 12.35) even though nothing was typed.
    draft.value =
      model.value === null || model.value === undefined ? '' : String(model.value);
    return;
  }
  const text = draft.value.trim();
  if (!text) {
    model.value = null;
    draft.value = '';
    edited.value = false;
    invalidDraft.value = false;
    return;
  }
  if (parseChoyNumber(draft.value, 'decimal') === null) {
    // Keep the invalid draft visible so the user can correct it, but mark the control
    // invalid so it cannot look committed while the host still holds the old amount.
    edited.value = true;
    invalidDraft.value = true;
    return;
  }
  const rounded = roundChoyDecimal(draft.value, resolveChoyMonetaryPrecision(props.precision));
  if (!rounded) {
    edited.value = true;
    invalidDraft.value = true;
    return;
  }
  model.value = rounded.value;
  draft.value = rounded.text;
  edited.value = false;
  invalidDraft.value = false;
}

function onInput(value: string): void {
  // Guard like datetime/time: some browsers still emit updates when readonly/disabled.
  if (props.readonly || props.disabled) {
    return;
  }
  edited.value = true;
  invalidDraft.value = false;
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
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <Input
        :id="controlId"
        :model-value="displayValue"
        :name="name || undefined"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        :aria-invalid="ariaInvalid || invalidDraft || undefined"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
        inputmode="decimal"
        @update:model-value="onInput"
        @focus="onFocus"
        @blur="onBlur"
      />
    </template>
  </ChoyFieldBase>
</template>
