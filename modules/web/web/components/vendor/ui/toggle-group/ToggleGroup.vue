<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { ToggleGroupRoot } from 'reka-ui';
import { provide } from 'vue';
import { cn, type ClassValue } from '../../../../lib/utils';

type ToggleVariant = 'default' | 'outline';
type ToggleSize = 'default' | 'sm' | 'lg';

const modelValue = defineModel<string | string[] | undefined>();

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    variant?: ToggleVariant;
    size?: ToggleSize;
    spacing?: number;
    type?: 'single' | 'multiple';
    disabled?: boolean;
    orientation?: 'horizontal' | 'vertical';
    loop?: boolean;
    rovingFocus?: boolean;
  }>(),
  {
    spacing: 0,
    type: 'single',
    disabled: false,
    orientation: 'horizontal',
    loop: true,
    rovingFocus: true,
  },
);

provide('toggleGroup', {
  variant: props.variant,
  size: props.size,
  spacing: props.spacing,
});
</script>

<template>
  <ToggleGroupRoot
    v-slot="slotProps"
    v-model="modelValue"
    data-slot="toggle-group"
    :type="type"
    :disabled="disabled"
    :orientation="orientation"
    :loop="loop"
    :roving-focus="rovingFocus"
    :data-size="size"
    :data-variant="variant"
    :data-spacing="spacing"
    :style="{ '--gap': spacing }"
    :class="
      cn(
        'group/toggle-group flex w-fit items-center gap-[--spacing(var(--gap))] rounded-md data-[spacing=default]:data-[variant=outline]:shadow-sm',
        props.class,
      )
    "
  >
    <slot v-bind="slotProps" />
  </ToggleGroupRoot>
</template>
