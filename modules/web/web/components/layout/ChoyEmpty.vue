<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Empty
    data-testid="choy-empty"
    :class="['choy-empty border-none p-6 md:p-8', props.class]"
  >
    <EmptyHeader>
      <EmptyTitle v-if="title">{{ title }}</EmptyTitle>
      <EmptyDescription v-if="description || $slots.default">
        <slot>{{ description }}</slot>
      </EmptyDescription>
    </EmptyHeader>
    <EmptyContent v-if="$slots.actions">
      <slot name="actions" />
    </EmptyContent>
  </Empty>
</template>

<script setup lang="ts">
import type { ClassValue } from '../../lib/utils';
import Empty from '../vendor/ui/empty/Empty.vue';
import EmptyContent from '../vendor/ui/empty/EmptyContent.vue';
import EmptyDescription from '../vendor/ui/empty/EmptyDescription.vue';
import EmptyHeader from '../vendor/ui/empty/EmptyHeader.vue';
import EmptyTitle from '../vendor/ui/empty/EmptyTitle.vue';

/**
 * Product empty-state chrome over vendor Empty*.
 */
const props = defineProps<{
  class?: ClassValue;
  title?: string;
  description?: string;
}>();
</script>
