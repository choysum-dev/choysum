<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Store hosts: bigint / int / decimal O* engines; chrome keeps defineModel. -->
  <OBigintField v-if="storeMode && mode === 'bigint'" v-bind="(storeBind as any)" />
  <OIntField v-else-if="storeMode && mode === 'integer'" v-bind="(storeBind as any)" />
  <ODecimalField v-else-if="storeMode" v-bind="(storeBind as any)" />
  <ChoyFieldBase
    v-else
    v-bind="($attrs as any)"
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
        :inputmode="mode === 'integer' || mode === 'bigint' ? 'numeric' : 'decimal'"
        @update:model-value="onDraftInput"
        @change="commitDraft"
        @blur="commitDraft"
        @keydown="onKeydown"
      />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { ref, useAttrs, watch } from 'vue';
import Input from '../vendor/ui/input/Input.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import OBigintField from './OBigintField.vue';
import ODecimalField from './ODecimalField.vue';
import OIntField from './OIntField.vue';
import {
  choyFieldChromeDefaults,
  parseChoyNumber,
  resolveChoyNumberDraftText,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

defineOptions({ name: 'ChoyNumberField', inheritAttrs: false });

type ChoyNumberMode = 'integer' | 'float' | 'decimal' | 'bigint';

/**
 * Numeric field. Store+prop hosts OInt / ODecimal / OBigint by `mode`.
 * Chrome model is `number | null` (bigint chrome is text-backed via draft only).
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      mode?: ChoyNumberMode;
      store?: WebModelStore<any>;
      prop?: string;
      binding?: unknown;
      rules?: unknown[];
      vColumnProps?: Record<string, unknown>;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: '',
    mode: 'float',
  },
);

const attrs = useAttrs();
const { storeMode, storeBind } = useChoyStoreFieldBinding(props as any, attrs as Record<string, unknown>);

const model = defineModel<number | null>({ default: null });
const draft = ref(
  model.value === null || model.value === undefined ? '' : String(model.value),
);
/** True when commit kept an unparsed draft while the host model is unchanged. */
const invalidDraft = ref(false);

/**
 * Host model is already a number — validate mode against the value itself.
 * `String(1e-22)` is exponential and `parseChoyNumber` rejects it, but the host
 * number is still a valid float/decimal.
 */
function isHostNumberCompatibleWithMode(value: number, mode: ChoyNumberMode): boolean {
  if (!Number.isFinite(value)) {
    return false;
  }
  if (mode === 'integer' || mode === 'bigint') {
    return Number.isSafeInteger(value);
  }
  return Number.isSafeInteger(Math.trunc(value));
}

/** Chrome parse mode collapses bigint onto integer digit rules. */
function chromeParseMode(mode: ChoyNumberMode): 'integer' | 'float' | 'decimal' {
  return mode === 'bigint' ? 'integer' : mode;
}

watch(model, (next) => {
  const expected = next === null || next === undefined ? '' : String(next);
  const parseMode = chromeParseMode(props.mode);
  const parsed = parseChoyNumber(draft.value, parseMode);
  const draftInvalid = draft.value.trim() !== '' && parsed === null;
  if (parsed !== next || draftInvalid) {
    draft.value = expected;
    invalidDraft.value =
      next !== null &&
      next !== undefined &&
      !isHostNumberCompatibleWithMode(next, props.mode);
  }
});

watch(
  () => props.mode,
  () => {
    // On mount or after a mode switch, show the host text and flag it when the
    // mode cannot accept that host number (e.g. model 12.5 under integer). Do
    // not rewrite the host — the user (or host) must correct it.
    const host = model.value;
    draft.value = host === null || host === undefined ? '' : String(host);
    invalidDraft.value =
      host !== null &&
      host !== undefined &&
      !isHostNumberCompatibleWithMode(host, props.mode);
  },
  { immediate: true },
);

function onDraftInput(value: string | number): void {
  // Guard like the monetary/datetime fields: some browsers still emit updates
  // while the underlying input is readonly or disabled.
  if (props.readonly || props.disabled) {
    return;
  }
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
  const parseMode = chromeParseMode(props.mode);
  const parsed = parseChoyNumber(draft.value, parseMode);
  if (parsed === null) {
    // Keep the draft so the user can correct invalid input; leave the model unchanged,
    // but mark the control invalid so it cannot look committed.
    invalidDraft.value = true;
    return;
  }
  invalidDraft.value = false;
  model.value = parsed;
  draft.value = resolveChoyNumberDraftText(parsed, text, parseMode);
}

function onKeydown(event: KeyboardEvent): void {
  // Confirming an IME candidate also fires Enter; don't commit half-composed text.
  if (event.key !== 'Enter' || event.isComposing || event.keyCode === 229) {
    return;
  }
  event.preventDefault();
  commitDraft();
}
</script>
