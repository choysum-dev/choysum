<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed } from 'vue';
import { SwitchRoot, SwitchThumb } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

const props = defineProps<{
  class?: ClassValue;
  disabled?: boolean;
  id?: string;
  'aria-invalid'?: boolean | 'true' | 'false';
  'aria-required'?: boolean | 'true' | 'false';
  'aria-describedby'?: string;
}>();

const checked = defineModel<boolean>({ default: false });

/** Checked chrome via Vue binding + tokens — avoids relying on data-[state=*]:* utilities in the served bundle. */
const rootStyle = computed(() => ({
  paddingInline: '2px',
  backgroundColor: checked.value ? 'var(--choy-color-primary)' : 'var(--choy-color-input)',
  borderColor: checked.value ? 'var(--choy-color-primary)' : 'var(--choy-color-border)',
}));
</script>

<template>
  <SwitchRoot
    :id="id"
    v-model="checked"
    data-slot="switch"
    :disabled="disabled"
    :aria-invalid="props['aria-invalid']"
    :aria-required="props['aria-required']"
    :aria-describedby="props['aria-describedby']"
    :class="
      cn(
        'peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border shadow-sm transition-colors',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
    :style="rootStyle"
  >
    <SwitchThumb
      data-slot="switch-thumb"
      :class="
        cn(
          'pointer-events-none block size-4 rounded-full bg-background shadow-sm ring-1 ring-border',
          checked ? 'ms-auto' : undefined,
        )
      "
    />
  </SwitchRoot>
</template>
