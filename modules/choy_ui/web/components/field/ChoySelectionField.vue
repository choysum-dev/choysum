<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, watch } from 'vue';
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

// Reka rejects '' as an item value; treat an empty string as "unset".
watch(
  model,
  (value) => {
    if (value === '') {
      model.value = null;
    }
  },
  { immediate: true },
);

/** Reka SelectItem rejects empty-string values and duplicates; null model covers "unset". */
const selectOptions = computed(() => {
  const seen = new Set<string>();
  const out: ChoySelectionOption[] = [];
  for (const opt of props.options ?? []) {
    if (opt.value === '' || seen.has(opt.value)) {
      continue;
    }
    seen.add(opt.value);
    out.push(opt);
  }
  return out;
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
    <template #default="{ controlId, ariaInvalid, ariaRequired, ariaDescribedby }">
      <Select v-model="model" :disabled="disabled || readonly">
        <SelectTrigger
          :id="controlId"
          :placeholder="placeholder"
          :disabled="disabled || readonly"
          :aria-invalid="ariaInvalid"
          :aria-required="ariaRequired"
          :aria-describedby="ariaDescribedby"
        />
        <SelectContent>
          <SelectItem
            v-for="opt in selectOptions"
            :key="opt.value"
            :value="opt.value"
          >
            {{ opt.label }}
          </SelectItem>
        </SelectContent>
      </Select>
    </template>
  </ChoyFieldBase>
</template>
