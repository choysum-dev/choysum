<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';

/**
 * Fires `visible` when this node intersects the lane scrollport (for auto load-more).
 * Must be rendered inside `.choy-kanban-view__lane-body`.
 *
 * After a fire, waits for a non-intersecting frame (or a short cooldown after busy
 * clears) before emitting again so load-more ↔ laneFetchBusy does not storm RPCs.
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

/** Delay before re-arming while still intersecting after busy clears (ms). */
const REARM_AFTER_BUSY_MS = 250;

const root = ref<HTMLElement | null>(null);
let observer: IntersectionObserver | null = null;
/** False after emit until leave-view or busy-clear cooldown. */
let armed = true;
let lastIntersecting = false;
let rearmTimer: ReturnType<typeof setTimeout> | null = null;

function clearRearmTimer(): void {
  if (rearmTimer != null) {
    clearTimeout(rearmTimer);
    rearmTimer = null;
  }
}

function disconnect(): void {
  observer?.disconnect();
  observer = null;
}

function fireVisible(): void {
  if (props.disabled || !armed) return;
  armed = false;
  emit('visible');
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
      const intersecting = entries.some((entry) => entry.isIntersecting);
      lastIntersecting = intersecting;
      if (!intersecting) {
        armed = true;
        return;
      }
      fireVisible();
    },
    { root: scrollRoot, rootMargin: '80px 0px', threshold: 0 },
  );
  observer.observe(el as unknown as Element);
}

onMounted(connect);
onBeforeUnmount(() => {
  clearRearmTimer();
  disconnect();
});

watch(
  () => props.disabled,
  (disabled, wasDisabled) => {
    if (disabled) {
      clearRearmTimer();
      return;
    }
    // Busy cleared: the existing observer already tracked whether the sentinel
    // stayed on screen. Re-binding would emit immediately and again after cooldown.
    if (wasDisabled && root.value) {
      clearRearmTimer();
      rearmTimer = setTimeout(() => {
        rearmTimer = null;
        armed = true;
        if (!props.disabled && lastIntersecting) {
          fireVisible();
        }
      }, REARM_AFTER_BUSY_MS);
    }
  },
);

defineExpose({
  /** Test hook: whether the next intersecting callback may emit. */
  isArmed: () => armed,
  /** Test hook: simulate IO intersection changes. */
  handleIntersectionForTest: (intersecting: boolean) => {
    lastIntersecting = intersecting;
    if (!intersecting) {
      armed = true;
      return;
    }
    fireVisible();
  },
});
</script>

<template>
  <div
    ref="root"
    class="choy-kanban-view__load-sentinel h-px w-full shrink-0"
    data-testid="choy-kanban-load-more-sentinel"
    aria-hidden="true"
  />
</template>
