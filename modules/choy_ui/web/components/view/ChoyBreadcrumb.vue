<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router';
import { ChevronRight } from 'lucide-vue-next';
import type { ClassValue } from '../../lib/utils';

export type ChoyBreadcrumbItem = {
  label: string;
  to?: RouteLocationRaw;
};

/**
 * Breadcrumb nav. Items with `to` render as router-link; others as span.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    items?: ChoyBreadcrumbItem[];
  }>(),
  {
    items: () => [],
  },
);
</script>

<template>
  <nav
    data-anchor="choy.breadcrumb"
    aria-label="Breadcrumb"
    :class="['choy-breadcrumb text-sm text-foreground/70', props.class]"
  >
    <ol class="flex flex-wrap items-center gap-1">
      <li
        v-for="(item, index) in items"
        :key="`${item.label}-${index}`"
        class="flex items-center gap-1"
      >
        <ChevronRight
          v-if="index > 0"
          class="size-3.5 shrink-0 text-foreground/40"
          aria-hidden="true"
        />
        <RouterLink
          v-if="item.to && index !== items.length - 1"
          :to="item.to"
          class="hover:text-foreground hover:underline"
        >
          {{ item.label }}
        </RouterLink>
        <span
          v-else
          class="text-foreground"
          :aria-current="index === items.length - 1 ? 'page' : undefined"
        >
          {{ item.label }}
        </span>
      </li>
    </ol>
  </nav>
</template>
