<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { computed, useSlots } from 'vue';
import Card from '../vendor/ui/card/Card.vue';
import CardContent from '../vendor/ui/card/CardContent.vue';
import CardFooter from '../vendor/ui/card/CardFooter.vue';
import CardHeader from '../vendor/ui/card/CardHeader.vue';
import CardTitle from '../vendor/ui/card/CardTitle.vue';
import type { ClassValue } from '../../lib/utils';

/**
 * Public section card. Optional title prop; or use header/default/footer slots.
 */
const props = defineProps<{
  class?: ClassValue;
  title?: string;
}>();

const slots = useSlots();
const hasHeader = computed(() => !!props.title || !!slots.header);
</script>

<template>
  <Card data-anchor="choy.card" :class="props.class">
    <CardHeader v-if="hasHeader">
      <slot name="header">
        <CardTitle v-if="title">{{ title }}</CardTitle>
      </slot>
    </CardHeader>
    <CardContent v-if="$slots.default" :class="hasHeader ? undefined : 'pt-4'">
      <slot />
    </CardContent>
    <CardFooter v-if="$slots.footer">
      <slot name="footer" />
    </CardFooter>
  </Card>
</template>
