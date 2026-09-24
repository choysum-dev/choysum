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

watch(model, (next) => {
  const expected = next === null || next === undefined ? '' : String(next);
  const parsed = parseChoyNumber(draft.value, props.mode);
  const draftInvalid = draft.value.trim() !== '' && parsed === null;
  if (parsed !== next || draftInvalid) {
    draft.value = expected;
  }
});

function commitDraft(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  const text = draft.value.trim();
  if (!text) {
    model.value = null;
    draft.value = '';
    return;
  }
  const parsed = parseChoyNumber(draft.value, props.mode);
  if (parsed === null) {
    // Keep the draft so the user can correct invalid input; leave the model unchanged.
    return;
  }
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
    :error="error"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <Input
        :id="controlId"
        v-model="draft"
        type="text"
        :name="name || undefined"
        :placeholder="placeholder"
        :disabled="disabled"
        :readonly="readonly"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
        :inputmode="mode === 'integer' ? 'numeric' : 'decimal'"
        @change="commitDraft"
        @blur="commitDraft"
      />
    </template>
  </ChoyFieldBase>
</template>
