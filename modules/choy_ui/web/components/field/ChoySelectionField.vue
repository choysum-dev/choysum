<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import Select from '../vendor/ui/select/Select.vue';
import SelectContent from '../vendor/ui/select/SelectContent.vue';
import SelectItem from '../vendor/ui/select/SelectItem.vue';
import SelectTrigger from '../vendor/ui/select/SelectTrigger.vue';
import type { ClassValue } from '../../lib/utils';
import ChoyFieldBase from './ChoyFieldBase.vue';
import {
  choyFieldChromeDefaults,
  type ChoyFieldChromeProps,
  type ChoySelectionOption,
} from './fieldHelpers';

/**
 * Single-select dropdown field.
 */
const props = withDefaults(
  defineProps<
    ChoyFieldChromeProps & {
      class?: ClassValue;
      placeholder?: string;
      options?: ChoySelectionOption[];
    }
  >(),
  {
    ...choyFieldChromeDefaults,
    placeholder: 'Select…',
    options: () => [],
  },
);

const model = defineModel<string | null>({ default: null });

/** Reka Select expects string; map null to empty sentinel. */
const selectValue = computed({
  get: () => model.value ?? '',
  set: (next: string) => {
    model.value = next === '' ? null : next;
  },
});
</script>

<template>
  <ChoyFieldBase
    data-anchor="choy.selection-field"
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
    <Select v-model="selectValue" :disabled="disabled || readonly">
      <SelectTrigger :placeholder="placeholder" :disabled="disabled || readonly" />
      <SelectContent>
        <SelectItem
          v-for="opt in options"
          :key="opt.value"
          :value="opt.value"
        >
          {{ opt.label }}
        </SelectItem>
      </SelectContent>
    </Select>
  </ChoyFieldBase>
</template>
