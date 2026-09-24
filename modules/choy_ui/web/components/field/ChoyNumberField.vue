<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ref, watch } from 'vue';
import Input from '../vendor/ui/input/Input.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  parseChoyNumber,
  resolveChoyNumberDraftText,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Numeric field. Model is `number | null`; input text is parsed on blur/commit.
 * Always uses a text input so partial/invalid entries are not coerced to ''.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      mode?: 'integer' | 'float' | 'decimal';
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: '',
    mode: 'float',
  },
);

const model = defineModel<number | null>({ default: null });
const draft = ref(
  model.value === null || model.value === undefined ? '' : String(model.value),
);
/** True when commit kept an unparsed draft while the host model is unchanged. */
const invalidDraft = ref(false);

watch(model, (next) => {
  const expected = next === null || next === undefined ? '' : String(next);
  const parsed = parseChoyNumber(draft.value, props.mode);
  const draftInvalid = draft.value.trim() !== '' && parsed === null;
  if (parsed !== next || draftInvalid) {
    draft.value = expected;
    invalidDraft.value = false;
  }
});

function onDraftInput(value: string | number): void {
  draft.value = String(value ?? '');
  invalidDraft.value = false;
}

function commitDraft(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  const hostText =
    model.value === null || model.value === undefined ? '' : String(model.value);
  // Focus/blur alone must not flag a host value whose shortest form is exponential
  // (e.g. `1e-22`) as invalid; only real edits can be unparseable.
  if (draft.value === hostText) {
    return;
  }
  const text = draft.value.trim();
  if (!text) {
    model.value = null;
    draft.value = '';
    invalidDraft.value = false;
    return;
  }
  const parsed = parseChoyNumber(draft.value, props.mode);
  if (parsed === null) {
    // Keep the draft so the user can correct invalid input; leave the model unchanged,
    // but mark the control invalid so it cannot look committed.
    invalidDraft.value = true;
    return;
  }
  invalidDraft.value = false;
  model.value = parsed;
  draft.value = resolveChoyNumberDraftText(parsed, text, props.mode);
}
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.number-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="error || (invalidDraft ? 'Invalid number' : '')"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <Input
        :id="controlId"
        :model-value="draft"
        type="text"
        :name="name || undefined"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        :aria-invalid="ariaInvalid || invalidDraft || undefined"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
        :inputmode="mode === 'integer' ? 'numeric' : 'decimal'"
        @update:model-value="onDraftInput"
        @change="commitDraft"
        @blur="commitDraft"
        @keydown.enter.prevent="commitDraft"
      />
    </template>
  </ChoyFieldBase>
</template>
