<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TabsContent } from 'reka-ui';
import { cn, type ClassValue } from '../../../lib/utils';

defineProps<{
  class?: ClassValue;
  value: string;
}>();
</script>

<template>
  <TabsContent
    data-slot="tabs-content"
    :value="value"
    :class="
      cn(
        'mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
        $props.class,
      )
    "
  >
    <slot />
  </TabsContent>
</template>
