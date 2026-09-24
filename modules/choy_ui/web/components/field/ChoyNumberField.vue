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
  parseChoyNumber,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Numeric field. Model is `number | null`; input text is parsed on blur/commit.
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
  if (parseChoyNumber(draft.value, props.mode) !== next) {
    draft.value = expected;
  }
});

const inputType = computed(() => (props.mode === 'integer' ? 'number' : 'text'));

function commitDraft(): void {
  if (props.readonly || props.disabled) {
    return;
  }
  const parsed = parseChoyNumber(draft.value, props.mode);
  model.value = parsed;
  draft.value = parsed === null ? '' : String(parsed);
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
    <Input
      :id="name || undefined"
      v-model="draft"
      :type="inputType"
      :name="name || undefined"
      :placeholder="placeholder"
      :disabled="disabled || readonly"
      :readonly="readonly"
      inputmode="decimal"
      @change="commitDraft"
      @blur="commitDraft"
    />
  </ChoyFieldBase>
</template>
