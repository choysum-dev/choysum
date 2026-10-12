<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Breadcrumb
    data-anchor="choy.breadcrumb"
    :class="['choy-breadcrumb', props.class]"
  >
    <BreadcrumbList>
      <template v-for="(seg, segIndex) in displaySegments" :key="seg.key">
        <BreadcrumbSeparator v-if="segIndex > 0" />
        <BreadcrumbItem>
          <BreadcrumbEllipsis
            v-if="seg.type === 'ellipsis'"
            data-testid="choy-breadcrumb-ellipsis"
          />
          <template v-else>
            <BreadcrumbLink
              v-if="seg.item.onSelect && !seg.isLast"
              as-child
            >
              <button
                type="button"
                data-testid="choy-breadcrumb-action"
                @click="seg.item.onSelect"
              >
                {{ seg.item.label }}
              </button>
            </BreadcrumbLink>
            <BreadcrumbLink
              v-else-if="seg.item.to && !seg.isLast"
              as-child
            >
              <RouterLink :to="seg.item.to">
                {{ seg.item.label }}
              </RouterLink>
            </BreadcrumbLink>
            <!-- Intermediate non-link segments must not use BreadcrumbPage (aria-current=page). -->
            <span v-else-if="!seg.isLast">
              {{ seg.item.label }}
            </span>
            <BreadcrumbPage v-else>
              {{ seg.item.label }}
            </BreadcrumbPage>
          </template>
        </BreadcrumbItem>
      </template>
    </BreadcrumbList>
  </Breadcrumb>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { RouterLink, type RouteLocationRaw } from 'vue-router';
import type { ClassValue } from '../../lib/utils';
import Breadcrumb from '../vendor/ui/breadcrumb/Breadcrumb.vue';
import BreadcrumbEllipsis from '../vendor/ui/breadcrumb/BreadcrumbEllipsis.vue';
import BreadcrumbItem from '../vendor/ui/breadcrumb/BreadcrumbItem.vue';
import BreadcrumbLink from '../vendor/ui/breadcrumb/BreadcrumbLink.vue';
import BreadcrumbList from '../vendor/ui/breadcrumb/BreadcrumbList.vue';
import BreadcrumbPage from '../vendor/ui/breadcrumb/BreadcrumbPage.vue';
import BreadcrumbSeparator from '../vendor/ui/breadcrumb/BreadcrumbSeparator.vue';

export type ChoyBreadcrumbItem = {
  label: string;
  to?: RouteLocationRaw;
  /** Custom activation (e.g. L1 app → navigateTo(app)); preferred over `to` when set. */
  onSelect?: () => void;
};

type DisplaySegment =
  | { type: 'ellipsis'; key: string }
  | { type: 'item'; key: string; item: ChoyBreadcrumbItem; isLast: boolean };

/**
 * Breadcrumb nav over vendor Breadcrumb* (shadcn / sidebar-09 look).
 * List owns muted foreground; links use kit hover:text-foreground (no underline).
 * Trails longer than `maxItems` collapse the middle to BreadcrumbEllipsis
 * (keep first + last two, matching the shadcn docs demo).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    items?: ChoyBreadcrumbItem[];
    /** Visible crumb slots before collapsing middle (default 4 → first + … + last two). */
    maxItems?: number;
  }>(),
  {
    items: () => [],
    maxItems: 4,
  },
);

const displaySegments = computed((): DisplaySegment[] => {
  const items = props.items || [];
  const n = items.length;
  if (!n) return [];

  const maxItems = Math.max(3, props.maxItems || 4);
  const lastIndex = n - 1;

  const toItem = (item: ChoyBreadcrumbItem, index: number): DisplaySegment => ({
    type: 'item',
    key: `item-${index}-${item.label}`,
    item,
    isLast: index === lastIndex,
  });

  if (n <= maxItems) {
    return items.map((item, index) => toItem(item, index));
  }

  // Keep first + last two (shadcn docs / sidebar-style collapse).
  const tailStart = n - 2;
  const segs: DisplaySegment[] = [toItem(items[0]!, 0), { type: 'ellipsis', key: 'ellipsis' }];
  for (let i = tailStart; i < n; i++) {
    segs.push(toItem(items[i]!, i));
  }
  return segs;
});
</script>
