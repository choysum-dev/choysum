<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { Check } from 'lucide-vue-next';
import { CheckboxIndicator, CheckboxRoot } from 'reka-ui';
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
        'peer size-4 shrink-0 rounded-sm border border-border bg-background shadow-sm',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        'disabled:cursor-not-allowed disabled:opacity-50',
        'data-[state=checked]:bg-primary data-[state=checked]:text-background data-[state=indeterminate]:bg-primary',
        props.class,
      )
    "
  >
    <CheckboxIndicator class="flex items-center justify-center text-current">
      <Check class="size-3.5" />
    </CheckboxIndicator>
  </CheckboxRoot>
</template>
