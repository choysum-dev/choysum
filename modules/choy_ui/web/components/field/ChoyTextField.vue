<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import Textarea from '../vendor/ui/textarea/Textarea.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
} from './fieldHelpers';

/**
 * Multi-line string field.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      rows?: number;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: '',
    rows: 4,
  },
);

const model = defineModel<string>({ default: '' });
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.text-field"
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
      <Textarea
        :id="controlId"
        v-model="model"
        :name="name || undefined"
        :placeholder="placeholder"
        :rows="rows"
        :disabled="disabled"
        :readonly="readonly"
        :aria-invalid="ariaInvalid"
        :aria-describedby="ariaDescribedby"
      />
    </template>
  </ChoyFieldBase>
</template>
