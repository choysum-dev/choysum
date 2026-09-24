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
 * Single-line string field.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: '',
  },
);

const model = defineModel<string>({ default: '' });
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.varchar-field"
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
        v-model="model"
        :name="name || undefined"
        :placeholder="placeholder"
        :disabled="disabled || readonly"
        :readonly="readonly"
        :aria-invalid="ariaInvalid"
        :aria-describedby="ariaDescribedby"
      />
    </template>
  </ChoyFieldBase>
</template>
