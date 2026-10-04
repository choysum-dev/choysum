<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Card
    data-anchor="choy.card"
    :class="cn('flex flex-col gap-6 rounded-xl py-6', props.class)"
  >
    <div
      v-if="title || description || $slots.header"
      data-slot="card-header"
      class="grid auto-rows-min grid-rows-[auto_auto] items-start gap-1.5 px-6"
    >
      <slot name="header">
        <CardTitle v-if="title">{{ title }}</CardTitle>
        <CardDescription v-if="description" class="text-muted-foreground">{{ description }}</CardDescription>
      </slot>
    </div>
    <div v-if="$slots.default" data-slot="card-content" class="px-6">
      <slot />
    </div>
    <div v-if="$slots.footer" data-slot="card-footer" class="flex items-center px-6">
      <slot name="footer" />
    </div>
  </Card>
</template>

<script setup lang="ts">
import Card from '../vendor/ui/card/Card.vue';
import CardDescription from '../vendor/ui/card/CardDescription.vue';
import CardTitle from '../vendor/ui/card/CardTitle.vue';
import { cn, type ClassValue } from '../../lib/utils';

/**
 * Public section card. Optional title and description; or use header/default/footer slots.
 * Header visibility reads `$slots` at render time (slots are not reactive).
 * Spacing follows shadcn-vue login-01 / new-york-v4 (gap-6, px-6, py-6, rounded-xl).
 */
const props = defineProps<{
  class?: ClassValue;
  title?: string;
  description?: string;
}>();
</script>
