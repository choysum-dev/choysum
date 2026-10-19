<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { Check } from 'lucide-vue-next';
import { CheckboxIndicator, CheckboxRoot } from 'reka-ui';
import { computed } from 'vue';
import { cn, type ClassValue } from '../../../../lib/utils';

const props = defineProps<{
  class?: ClassValue;
  disabled?: boolean;
  id?: string;
  'aria-invalid'?: boolean | 'true' | 'false';
  'aria-required'?: boolean | 'true' | 'false';
  'aria-describedby'?: string;
}>();

const checked = defineModel<boolean | 'indeterminate'>({ default: false });

const on = computed(() => checked.value === true || checked.value === 'indeterminate');
</script>

<template>
  <CheckboxRoot
    :id="id"
    v-model="checked"
    data-slot="checkbox"
    :disabled="disabled"
    :aria-invalid="props['aria-invalid']"
    :aria-required="props['aria-required']"
    :aria-describedby="props['aria-describedby']"
    :class="
      cn(
        'peer inline-flex size-4 shrink-0 items-center justify-center self-center appearance-none rounded-sm border p-0 leading-none shadow-xs outline-none',
        'focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50',
        'disabled:cursor-not-allowed disabled:opacity-50',
        on
          ? 'border-primary bg-primary text-primary-foreground'
          : 'border-input bg-transparent',
        props.class,
      )
    "
  >
    <CheckboxIndicator class="flex items-center justify-center text-current">
      <Check class="size-3.5 shrink-0" />
    </CheckboxIndicator>
  </CheckboxRoot>
</template>
