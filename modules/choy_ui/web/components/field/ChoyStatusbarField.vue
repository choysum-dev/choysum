<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
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

/** Drop empty values so an unset model cannot look pressed (parity with selection). */
const statusOptions = computed(() =>
  (props.options ?? []).filter((opt) => typeof opt.value === 'string' && opt.value !== ''),
);

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
    <template #default="{ controlId, labelId, ariaInvalid, ariaDescribedby }">
      <div
        :id="controlId"
        class="choy-statusbar-field flex flex-wrap gap-1"
        role="group"
        :aria-labelledby="label ? labelId : undefined"
        :aria-label="label ? undefined : name || 'Status'"
        :aria-invalid="ariaInvalid"
        :aria-describedby="ariaDescribedby"
        :aria-disabled="disabled || readonly || undefined"
      >
        <ChoyButton
          v-for="(opt, index) in statusOptions"
          :key="`${opt.value}-${index}`"
          type="button"
          size="sm"
          :variant="model === opt.value ? 'default' : 'outline'"
          :disabled="disabled || readonly"
          :aria-pressed="model === opt.value"
          @click="select(opt.value)"
        >
          {{ opt.label }}
        </ChoyButton>
      </div>
    </template>
  </ChoyFieldBase>
</template>
