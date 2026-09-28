<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Store hosts: bigint / int / float(number) / decimal engines; chrome keeps defineModel. -->
  <BigintField v-if="storeMode && mode === 'bigint'" v-bind="(storeBind as any)" />
  <IntField v-else-if="storeMode && mode === 'integer'" v-bind="(storeBind as any)" />
  <NumberField v-else-if="storeMode && mode === 'float'" v-bind="(storeBind as any)" />
  <DecimalField v-else-if="storeMode" v-bind="(storeBind as any)" />
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
        :inputmode="chromeInputMode"
        @update:model-value="onDraftInput"
        @change="commitDraft"
        @blur="commitDraft"
        @keydown="onKeydown"
      />
    </template>
  </ChoyFieldBase>
</template>

<script setup lang="ts">
import { computed, ref, useAttrs, watch } from 'vue';
import Input from '../vendor/ui/input/Input.vue';
import type { ClassValue } from '../../lib/utils';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import ChoyFieldBase from './ChoyFieldBase.vue';
import BigintField from './BigintField.vue';
import DecimalField from './DecimalField.vue';
import IntField from './IntField.vue';
import NumberField from './NumberField.vue';
import {
  choyFieldChromeDefaults,
  parseChoyNumber,
  resolveChoyNumberDraftText,
  type ChoyFieldChromeProps,
} from './fieldHelpers';
import {
  isChoyNumberHostCompatibleWithMode,
  resolveChoyNumberChromeParseMode,
  resolveChoyNumberInputMode,
  type ChoyNumberMode,
} from './choyNumberFieldChrome';

defineOptions({ name: 'ChoyNumberField', inheritAttrs: false });

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
    mode: 'decimal',
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

const chromeInputMode = computed(() => resolveChoyNumberInputMode(props.mode));

watch(model, (next) => {
  const expected = next === null || next === undefined ? '' : String(next);
  const parseMode = resolveChoyNumberChromeParseMode(props.mode);
  const parsed = parseChoyNumber(draft.value, parseMode);
  const draftInvalid = draft.value.trim() !== '' && parsed === null;
  if (parsed !== next || draftInvalid) {
    draft.value = expected;
    invalidDraft.value =
      next !== null &&
      next !== undefined &&
      !isChoyNumberHostCompatibleWithMode(next, props.mode);
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
      !isChoyNumberHostCompatibleWithMode(host, props.mode);
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
  const parseMode = resolveChoyNumberChromeParseMode(props.mode);
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
