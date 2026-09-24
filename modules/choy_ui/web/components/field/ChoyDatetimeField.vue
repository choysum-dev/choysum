<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import Input from '../vendor/ui/input/Input.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Datetime field via native datetime-local input (DatePicker is date-only).
 * Model is a string (typically `YYYY-MM-DDTHH:mm`) or null.
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

function onInput(value: string): void {
  model.value = value === '' ? null : value;
}
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.datetime-field"
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
    <template #default="{ controlId, ariaInvalid, ariaDescribedby }">
      <Input
        :id="controlId"
        type="datetime-local"
        :model-value="model ?? ''"
        :name="name || undefined"
        :disabled="disabled"
        :readonly="readonly"
        :aria-invalid="ariaInvalid"
        :aria-describedby="ariaDescribedby"
        @update:model-value="onInput"
      />
    </template>
  </ChoyFieldBase>
</template>
