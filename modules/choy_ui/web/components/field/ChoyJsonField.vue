<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';
import {
  normalizeChoyJsonIncoming,
  stringifyChoyJson,
  tryParseChoyJson,
  type ChoyJsonValue,
} from './jsonFieldHelpers';

/**
 * JSON object (optional array) field. Edit via textarea; display via pretty &lt;pre&gt;.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      nullable?: boolean;
      allowArray?: boolean;
      rows?: number;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: 'Enter a JSON object',
    nullable: true,
    allowArray: false,
    rows: 8,
  },
);

const model = defineModel<ChoyJsonValue>({ default: null });

const draft = ref(stringifyChoyJson(normalizeChoyJsonIncoming(model.value)));
const parseError = ref('');

watch(
  () => model.value,
  next => {
    const pretty = stringifyChoyJson(normalizeChoyJsonIncoming(next));
    const check = tryParseChoyJson(draft.value, {
      allowArray: props.allowArray,
      nullable: props.nullable,
    });
    // Keep an in-progress invalid draft; only sync when the textarea already matches the model.
    if (!check.ok) return;
    if (stringifyChoyJson(check.value) === pretty) return;
    draft.value = pretty;
    parseError.value = '';
  },
);

const displayText = computed(() => stringifyChoyJson(normalizeChoyJsonIncoming(model.value)));

const fieldError = computed(() => props.error || parseError.value);

function commitDraft(): void {
  const result = tryParseChoyJson(draft.value, {
    allowArray: props.allowArray,
    nullable: props.nullable,
  });
  if (!result.ok) {
    parseError.value = result.error;
    return;
  }
  parseError.value = '';
  model.value = result.value;
  draft.value = stringifyChoyJson(result.value);
}
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.json-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="required"
    :readonly="readonly"
    :disabled="disabled"
    :error="fieldError"
    :name="name"
    :visible="visible"
  >
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <pre
        v-if="readonly"
        :id="controlId"
        class="choy-json-field__display max-h-80 overflow-auto rounded-md border border-border bg-muted/20 p-3 text-xs"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
      >{{ displayText }}</pre>
      <Textarea
        v-else
        :id="controlId"
        v-model="draft"
        :name="name || undefined"
        :placeholder="placeholder"
        :rows="rows"
        :disabled="disabled"
        class="font-mono text-xs"
        :aria-invalid="ariaInvalid || !!parseError"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
        @blur="commitDraft"
      />
    </template>
  </ChoyFieldBase>
</template>
