<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        v-if="visible"
        data-anchor="choy.loading.slow-hint"
        role="status"
        aria-live="polite"
        :aria-label="plainMessage"
        class="choy-loading-slow-hint pointer-events-none fixed bottom-4 right-4 z-[100] flex max-w-sm items-start gap-2 rounded-md border border-border/60 bg-background/90 px-2.5 py-1.5 text-sm text-foreground shadow-sm backdrop-blur"
      >
        <ChoySpinner class="mt-0.5 size-3.5 shrink-0 text-primary" />
        <!-- Trusted product i18n (Odoo 7.0-style <br />); not user HTML. -->
        <span class="min-w-0 leading-5" v-html="message" />
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import {
  useHasActiveSurfaceBusy,
  useLoadingSlowHintVisible,
  useNavigationHandoffBusy,
} from '@/core/web/client/loadingRegistry';
import { createTranslate } from '@/web/web/i18n';
import ChoySpinner from './ChoySpinner.vue';

defineOptions({ name: 'ChoyLoadingSlowHint' });

const { _t } = createTranslate('web', { scope: 'web/components/layout/LoadingSlowHint' });

/** Registry slow-hint channel; suppressed while a full-viewport/surface block owns busy. */
const registryVisible = useLoadingSlowHintVisible();
const handoffBusy = useNavigationHandoffBusy();
const surfaceBusy = useHasActiveSurfaceBusy();
const visible = computed(
  () => registryVisible.value && !handoffBusy.value && !surfaceBusy.value,
);

/**
 * Odoo 7.0 Throbber `messages_by_seconds` (coresetup.js): seconds since the
 * slow hint became visible (our hint already waits ~3s of userBusy, like
 * BlockUI).
 */
const MESSAGE_STAGES: ReadonlyArray<readonly [number, string]> = [
  [0, 'Loading...'],
  [20, 'Still loading...'],
  [60, 'Still loading...<br />Please be patient.'],
  [120, "Don't leave yet,<br />it's still loading..."],
  [300, 'You may not believe it,<br />but the application is actually loading...'],
  [420, "Take a minute to get a coffee,<br />because it's loading..."],
  [3600, 'Maybe you should consider reloading the application by pressing F5...'],
];

const stageIndex = ref(0);
let shownAtMs = 0;
let tickTimer: ReturnType<typeof setInterval> | null = null;

function clearTick(): void {
  if (tickTimer != null) {
    clearInterval(tickTimer);
    tickTimer = null;
  }
}

function pickStageIndex(elapsedSec: number): number {
  let idx = 0;
  for (let i = 0; i < MESSAGE_STAGES.length; i++) {
    if (elapsedSec >= MESSAGE_STAGES[i]![0]) idx = i;
  }
  return idx;
}

function syncStage(): void {
  if (!visible.value || shownAtMs <= 0) {
    stageIndex.value = 0;
    return;
  }
  const elapsedSec = (Date.now() - shownAtMs) / 1000;
  stageIndex.value = pickStageIndex(elapsedSec);
}

watch(
  visible,
  show => {
    clearTick();
    if (!show) {
      shownAtMs = 0;
      stageIndex.value = 0;
      return;
    }
    shownAtMs = Date.now();
    syncStage();
    // Odoo Throbber re-evaluates about once per second.
    tickTimer = setInterval(syncStage, 1000);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  clearTick();
});

const message = computed(() => _t(MESSAGE_STAGES[stageIndex.value]![1]));

/** Screen-reader label without markup. */
const plainMessage = computed(() =>
  String(message.value || '')
    .replace(/<br\s*\/?>/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim(),
);
</script>
