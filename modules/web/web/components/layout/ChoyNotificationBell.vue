<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- Product/inbox mode: host the dual-stack ONotificationBell engine. -->
  <ONotificationBell v-if="useInboxEngine" />

  <div
    v-else
    data-anchor="choy.notification-bell"
    :class="cn('choy-notification-bell relative inline-flex', props.class)"
  >
    <Button
      variant="ghost"
      size="icon"
      type="button"
      :aria-label="ariaLabel"
      @click="emit('click')"
    >
      <Bell class="h-4 w-4" aria-hidden="true" />
    </Button>
    <Badge
      v-if="badgeText"
      aria-hidden="true"
      class="pointer-events-none absolute -right-1 -top-1 min-w-5 justify-center px-1 text-[10px]"
    >
      {{ badgeText }}
    </Badge>
    <span
      role="status"
      class="absolute h-px w-px overflow-hidden whitespace-nowrap opacity-0"
    >
      {{ unreadCount ? `${unreadCount} unread notifications` : '' }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { Bell } from 'lucide-vue-next';
import { computed } from 'vue';
import Badge from '../vendor/ui/badge/Badge.vue';
import Button from '../vendor/ui/button/Button.vue';
import { cn, type ClassValue } from '../../lib/utils';
import ONotificationBell from './ONotificationBell.vue';
import { isChoyNotificationInboxMode } from './choyNotificationBellMode';

/**
 * Notification bell: omit `count` to host ONotificationBell (inbox);
 * pass `count` for chrome-only badge demos (Gallery).
 */
const props = defineProps<{
  class?: ClassValue;
  /** When set (including 0), render chrome badge instead of the inbox engine. */
  count?: number;
  label?: string;
}>();

const emit = defineEmits<{
  click: [];
}>();

const useInboxEngine = computed(() => isChoyNotificationInboxMode(props.count));

const unreadCount = computed(() => {
  const count = Number(props.count);
  return Number.isFinite(count) && count > 0 ? Math.max(1, Math.ceil(count)) : 0;
});

const badgeText = computed(() =>
  unreadCount.value > 99 ? '99+' : unreadCount.value ? String(unreadCount.value) : '',
);

const ariaLabel = computed(() => {
  const label = props.label || 'Notifications';
  return unreadCount.value ? `${label} (${unreadCount.value} unread)` : label;
});
</script>
