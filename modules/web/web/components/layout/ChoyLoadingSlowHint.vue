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
        <!-- Trusted product i18n; split on &lt;br /&gt; so we never bind raw HTML. -->
        <span class="min-w-0 leading-5">
          <template v-for="(line, i) in messageLines" :key="i">
            <br v-if="i > 0" />
            {{ line }}
          </template>
        </span>
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
/** Seconds since slow-hint became visible → stage index (Odoo Throbber thresholds). */
const MESSAGE_STAGE_SECONDS: ReadonlyArray<number> = [0, 20, 60, 120, 300, 420, 3600];

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
  for (let i = 0; i < MESSAGE_STAGE_SECONDS.length; i++) {
    if (elapsedSec >= MESSAGE_STAGE_SECONDS[i]!) idx = i;
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
      syncStage();
      return;
    }
    shownAtMs = Date.now();
    syncStage();
    // Odoo Throbber re-evaluates about once per second; stop once the last stage sticks.
    tickTimer = setInterval(() => {
      syncStage();
      if (stageIndex.value >= MESSAGE_STAGE_SECONDS.length - 1) clearTick();
    }, 1000);
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  clearTick();
});

const message = computed(() => {
  // Literal _t(...) so extract sees live msgids (runtime table lookup alone goes obsolete).
  switch (stageIndex.value) {
    case 1:
      return _t('Still loading...');
    case 2:
      return _t('Still loading...<br />Please be patient.');
    case 3:
      return _t("Don't leave yet,<br />it's still loading...");
    case 4:
      return _t('You may not believe it,<br />but the application is actually loading...');
    case 5:
      return _t("Take a minute to get a coffee,<br />because it's loading...");
    case 6:
      return _t('Maybe you should consider reloading the application by pressing F5...');
    default:
      return _t('Loading...');
  }
});

/** Lines from msgid that uses Odoo-style `<br />` separators (not rendered as HTML). */
const messageLines = computed(() =>
  String(message.value || '')
    .split(/<br\s*\/?>/i)
    .map(s => s.trim())
    .filter(Boolean),
);

/** Screen-reader label without markup. */
const plainMessage = computed(() => messageLines.value.join(' '));

defineExpose({
  /** FE coverage: drive literal `_t` stage branches without waiting real wall clocks. */
  setStageIndexForTests(index: number) {
    // Stop the throbber tick so syncStage cannot overwrite the forced index.
    clearTick();
    shownAtMs = 0;
    stageIndex.value = index;
  },
  getPlainMessageForTests() {
    return plainMessage.value;
  },
});
</script>
