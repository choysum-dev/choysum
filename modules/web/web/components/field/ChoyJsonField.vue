<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OJsonobjectField v-if="storeMode" v-bind="(storeBind as any)" />
  <ChoyFieldBase
    v-else
    v-bind="($attrs as any)"
    data-anchor="choy.json-field"
    :class="props.class"
    :label="label"
    :help="help"
    :required="!!required"
    :readonly="!!readonly"
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

<script setup lang="ts">
import { computed, ref, useAttrs, watch } from 'vue';
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { isChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import OJsonobjectField from './OJsonobjectField.vue';
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

defineOptions({ name: 'ChoyJsonField', inheritAttrs: false });

/**
 * JSON field. Store+prop hosts OJsonobjectField; otherwise textarea chrome.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      nullable?: boolean;
      store?: WebModelStore<any>;
      prop?: string;
      binding?: unknown;
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

const attrs = useAttrs();
const storeMode = computed(() => isChoyStoreFieldBinding(props));
const storeBind = computed(() => ({ ...attrs, ...props }) as any);

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
    // Only skip when the draft already reflects the incoming model; an external
    // model update (e.g. form reset) must win even if the draft is invalid.
    if (check.ok && stringifyChoyJson(check.value) === pretty) return;
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
