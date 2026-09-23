<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { Bell } from 'lucide-vue-next';
import { computed } from 'vue';
import Badge from '../vendor/ui/badge/Badge.vue';
import Button from '../vendor/ui/button/Button.vue';
import type { ClassValue } from '../../lib/utils';

/**
 * Simplified notification bell. No inbox API in PR3 — optional count badge only.
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    count?: number;
    label?: string;
  }>(),
  {
    count: 0,
    label: 'Notifications',
  },
);

const emit = defineEmits<{
  click: [];
}>();

const badgeText = computed(() => {
  const count = Number(props.count);
  if (!Number.isFinite(count) || count <= 0) {
    return '';
  }
  const whole = Math.max(1, Math.ceil(count));
  return whole > 99 ? '99+' : String(whole);
});

const ariaLabel = computed(() =>
  badgeText.value ? `${props.label} (${badgeText.value} unread)` : props.label,
);
</script>

<template>
  <div
    data-anchor="choy.notification-bell"
    :class="['choy-notification-bell relative inline-flex', props.class]"
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
  </div>
</template>
