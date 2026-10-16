<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';

/**
 * Fires `visible` when this node intersects the lane scrollport (for auto load-more).
 * Must be rendered inside `.choy-kanban-view__lane-body`.
 */
const props = withDefaults(
  defineProps<{
    disabled?: boolean;
  }>(),
  { disabled: false },
);

const emit = defineEmits<{
  visible: [];
}>();

const root = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;

function disconnect(): void {
  observer?.disconnect();
  observer = null;
}

function connect(): void {
  disconnect();
  const el = root.value;
  if (!el || typeof IntersectionObserver === 'undefined') return;
  const scrollRoot =
    (el.closest('.choy-kanban-view__lane-body') as HTMLElement | null) ||
    (el.closest('.choy-kanban-view__board--flat') as HTMLElement | null) ||
    null;
  observer = new IntersectionObserver(
    (entries) => {
      if (props.disabled) return;
      if (entries.some((entry) => entry.isIntersecting)) {
        emit('visible');
      }
    },
    { root: scrollRoot, rootMargin: '80px 0px', threshold: 0 },
  );
  observer.observe(el);
}

onMounted(connect);
onBeforeUnmount(disconnect);
watch(
  () => props.disabled,
  () => {
    // Re-check when busy clears while the sentinel stays in view.
    if (!props.disabled && root.value) {
      connect();
    }
  },
);
</script>

<template>
  <div
    ref="root"
    class="choy-kanban-view__load-sentinel h-px w-full shrink-0"
    data-testid="choy-kanban-load-more-sentinel"
    aria-hidden="true"
  />
</template>
