<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Breadcrumb
    data-anchor="choy.breadcrumb"
    :class="['choy-breadcrumb text-sm text-foreground/70', props.class]"
  >
    <BreadcrumbList>
      <template v-for="(item, index) in items" :key="`${item.label}-${index}`">
        <BreadcrumbSeparator v-if="index > 0" />
        <BreadcrumbItem>
          <BreadcrumbLink
            v-if="item.to && index !== items.length - 1"
            as-child
          >
            <RouterLink :to="item.to" class="hover:text-foreground hover:underline">
              {{ item.label }}
            </RouterLink>
          </BreadcrumbLink>
          <!-- Intermediate non-link segments must not use BreadcrumbPage (aria-current=page). -->
          <span
            v-else-if="index !== items.length - 1"
            class="text-foreground/70"
          >
            {{ item.label }}
          </span>
          <BreadcrumbPage v-else>
            {{ item.label }}
          </BreadcrumbPage>
        </BreadcrumbItem>
      </template>
    </BreadcrumbList>
  </Breadcrumb>
</template>

<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router';
import type { ClassValue } from '../../lib/utils';
import Breadcrumb from '../vendor/ui/breadcrumb/Breadcrumb.vue';
import BreadcrumbItem from '../vendor/ui/breadcrumb/BreadcrumbItem.vue';
import BreadcrumbLink from '../vendor/ui/breadcrumb/BreadcrumbLink.vue';
import BreadcrumbList from '../vendor/ui/breadcrumb/BreadcrumbList.vue';
import BreadcrumbPage from '../vendor/ui/breadcrumb/BreadcrumbPage.vue';
import BreadcrumbSeparator from '../vendor/ui/breadcrumb/BreadcrumbSeparator.vue';

export type ChoyBreadcrumbItem = {
  label: string;
  to?: RouteLocationRaw;
};

/**
 * Breadcrumb nav over vendor Breadcrumb*.
 * Only the last item uses BreadcrumbPage (aria-current="page"); intermediates
 * with `to` are links, without `to` are plain text.
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
