<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { TabsContent } from 'reka-ui';
import { cn, type ClassValue } from '../../../../lib/utils';

withDefaults(
  defineProps<{
    class?: ClassValue;
    value: string;
    /** Keep inactive pane mounted (e.g. Form field registration across O2M table/kanban tabs). */
    forceMount?: boolean;
  }>(),
  {
    forceMount: false,
  },
);
</script>

<template>
  <TabsContent
    data-slot="tabs-content"
    :value="value"
    :force-mount="forceMount || undefined"
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
