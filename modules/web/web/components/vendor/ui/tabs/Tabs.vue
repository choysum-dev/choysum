<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TabsRoot } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

const props = defineProps<{
  class?: ClassValue;
  defaultValue?: string;
}>();

const modelValue = defineModel<string>();
</script>

<template>
  <TabsRoot
    v-model="modelValue"
    data-slot="tabs"
    :default-value="defaultValue"
    :class="cn('flex flex-col gap-2', props.class)"
  >
    <slot />
  </TabsRoot>
</template>
