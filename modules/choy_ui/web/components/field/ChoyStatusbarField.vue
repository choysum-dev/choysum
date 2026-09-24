<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import ChoyButton from '../layout/ChoyButton.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
  type ChoySelectionOption,
} from './fieldHelpers';

/**
 * Horizontal status button group. Selected option uses the primary variant.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      options?: ChoySelectionOption[];
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    options: () => [],
  },
);

const model = defineModel<string>({ default: '' });

function select(value: string): void {
  if (props.readonly || props.disabled) {
    return;
  }
  model.value = value;
}
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.statusbar-field"
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
    <div
      class="choy-statusbar-field flex flex-wrap gap-1"
      role="group"
      :aria-disabled="disabled || readonly || undefined"
    >
      <ChoyButton
        v-for="opt in options"
        :key="opt.value"
        type="button"
        size="sm"
        :variant="model === opt.value ? 'default' : 'outline'"
        :disabled="disabled || readonly"
        @click="select(opt.value)"
      >
        {{ opt.label }}
      </ChoyButton>
    </div>
  </ChoyFieldBase>
</template>
