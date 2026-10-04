<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Card data-anchor="choy.card" :class="props.class">
    <CardHeader v-if="title || description || $slots.header">
      <slot name="header">
        <CardTitle v-if="title">{{ title }}</CardTitle>
        <CardDescription v-if="description">{{ description }}</CardDescription>
      </slot>
    </CardHeader>
    <CardContent v-if="$slots.default">
      <slot />
    </CardContent>
    <CardFooter v-if="$slots.footer">
      <slot name="footer" />
    </CardFooter>
  </Card>
</template>

<script setup lang="ts">
import Card from '../vendor/ui/card/Card.vue';
import CardContent from '../vendor/ui/card/CardContent.vue';
import CardDescription from '../vendor/ui/card/CardDescription.vue';
import CardFooter from '../vendor/ui/card/CardFooter.vue';
import CardHeader from '../vendor/ui/card/CardHeader.vue';
import CardTitle from '../vendor/ui/card/CardTitle.vue';
import type { ClassValue } from '../../lib/utils';

/**
 * Public section card. Optional title and description; or use header/default/footer slots.
 * Header visibility reads `$slots` at render time (slots are not reactive).
 */
const props = defineProps<{
  class?: ClassValue;
  title?: string;
  description?: string;
}>();
</script>
