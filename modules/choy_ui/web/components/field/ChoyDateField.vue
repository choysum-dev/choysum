<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import DatePicker from '../internal/DatePicker.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Date field (YYYY-MM-DD) wrapping the L3 DatePicker.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      clearable?: boolean;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: 'Pick a date',
    clearable: true,
  },
);

const model = defineModel<string | null>({ default: null });
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.date-field"
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
      <DatePicker
        v-model="model"
        :id="controlId"
        :placeholder="placeholder"
        :disabled="disabled || readonly"
        :clearable="clearable && !readonly"
        :aria-invalid="ariaInvalid"
        :aria-required="ariaRequired"
        :aria-describedby="ariaDescribedby"
      />
    </template>
  </ChoyFieldBase>
</template>
