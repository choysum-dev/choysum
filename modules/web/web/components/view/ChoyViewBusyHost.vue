<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    data-slot="view-body-host"
    :class="hostClass"
    v-bind="hostAttrs"
  >
    <!-- Soft-stale: keep prior frame; tell the user refresh failed. -->
    <div
      v-if="showStaleBanner"
      data-testid="choy-view-surface-stale-banner"
      role="status"
      class="flex shrink-0 items-center justify-between gap-3 border-b border-border/60 bg-muted/40 px-3 py-2 text-sm text-foreground"
    >
      <span class="min-w-0 leading-5">{{ staleBannerCopy }}</span>
      <ChoyButton
        type="button"
        size="sm"
        variant="outline"
        data-testid="choy-view-surface-stale-retry"
        @click="emit('retry')"
      >
        {{ retryLabel }}
      </ChoyButton>
    </div>
    <!--
      Always keep the slot mounted so Action chrome under the host (table headers,
      board columns) does not unmount on no-frame failure / Retry.
    -->
    <div
      class="flex min-h-0 min-w-0 flex-1 flex-col"
      :inert="slotInert || undefined"
    >
      <slot />
    </div>
    <!--
      No-frame Action failure: overlay on top of chrome, do not replace the slot.
      Asset blank pages stay on ChoyViewLoadError (not this host).
    -->
    <ChoyViewSurfaceError
      v-if="showSurfaceError"
      class="absolute inset-0 z-20 bg-background/90"
      :title="surfaceCopy.title"
      :detail="surfaceCopy.detail"
      @retry="emit('retry')"
    />
    <!-- Live region must stay outside the inert host subtree so AT can announce it. -->
    <Teleport to="body">
      <div
        v-if="busy"
        role="status"
        aria-live="polite"
        class="sr-only"
        data-slot="view-busy-status"
      >
        {{ busyLabel }}
      </div>
    </Teleport>
    <!-- stale: light fade via host opacity only; no blocking overlay. -->
    <Transition name="busy-overlay">
      <div
        v-if="showOverlay"
        data-slot="view-busy-overlay"
        :data-mode="mode"
        class="absolute inset-0 z-10"
        :class="
          mode === 'skeleton'
            ? 'flex flex-col gap-2 bg-background/80 p-4'
            : 'flex items-center justify-center bg-background/50'
        "
        aria-hidden="true"
      >
        <template v-if="mode === 'skeleton'">
          <ChoySkeleton class="h-8 w-1/3" />
          <ChoySkeleton class="h-10 w-full" />
          <ChoySkeleton class="h-10 w-full" />
          <ChoySkeleton class="h-10 w-2/3" />
        </template>
        <ChoySpinner v-else class="size-6 text-primary" />
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue';
import ChoyButton from '../layout/ChoyButton.vue';
import ChoySkeleton from '../layout/ChoySkeleton.vue';
import ChoySpinner from '../layout/ChoySpinner.vue';
import type { ClassValue } from '../../lib/utils';
import { createTranslate } from '@/web/web/i18n';
import { useDelayedVisualBusy } from '@/web/web/composables/useDelayedVisualBusy';
import {
  resolveViewSurfaceErrorCopy,
  resolveViewSurfaceStaleBannerCopy,
} from '@/web/web/composables/resolveViewSurfaceErrorCopy';
import { useConnectivityStore } from '../../stores/connectivityStore';
import { VIEW_LOAD_RETRY_AFTER_RECOVERY_MS } from '../../router/viewLoadError';
import { viewBusyHostAttrs, viewBusyHostClass, type ViewBusyMode } from './viewBusy';
import ChoyViewSurfaceError from './ChoyViewSurfaceError.vue';

defineOptions({ name: 'ChoyViewBusyHost' });

const { _t } = createTranslate('web', { scope: 'web/components/view/ViewBusyHost' });
const connectivity = useConnectivityStore();

const props = withDefaults(
  defineProps<{
    busy?: boolean;
    /** stale | skeleton | block; default block keeps prior lock+spinner behavior. */
    mode?: ViewBusyMode;
    class?: ClassValue;
    /** Polite live-region text announced while busy (overlay stays aria-hidden). */
    busyLabel?: string;
    /** Controller/surface load error; with hasSettled and !hasFrame → surface failure UI. */
    error?: unknown;
    /** True when the surface already has a successful content frame. */
    hasFrame?: boolean;
    /** True after at least one load attempt finished (success or failure). */
    hasSettled?: boolean;
    /** Optional non-outage title override for resolveViewSurfaceErrorCopy. */
    errorFallbackTitle?: string;
  }>(),
  { busy: false, mode: 'block', hasFrame: false, hasSettled: false },
);

const emit = defineEmits<{
  retry: [];
}>();

const busy = computed(() => !!props.busy);
const mode = computed<ViewBusyMode>(() => props.mode || 'block');
const busyLabel = computed(() => props.busyLabel || _t('Loading'));
const retryLabel = computed(() => _t('Retry'));
/** Overlay only — aria-busy / inert stay on logical `busy`. Stale never uses an overlay. */
const overlayCandidate = computed(() => busy.value && mode.value !== 'stale');
const delayedOverlay = useDelayedVisualBusy(overlayCandidate);
/**
 * No-frame Retry clears error before the delayed busy overlay paints — show the
 * busy cover immediately so table chrome never flashes between error and spinner.
 */
const showOverlay = computed(
  () => overlayCandidate.value && (!props.hasFrame || delayedOverlay.value),
);
const hostClass = computed(() => viewBusyHostClass(busy.value, mode.value, props.class));
const hostAttrs = computed(() => viewBusyHostAttrs(busy.value, mode.value));

const showSurfaceError = computed(
  () => !!props.hasSettled && props.error != null && !props.hasFrame && !busy.value,
);

const showStaleBanner = computed(
  () => !!props.hasSettled && props.error != null && !!props.hasFrame && !busy.value,
);

/** Any Action failure UI that already exposes Retry — soft-retry after reconnect. */
const showRetryableFailure = computed(() => showSurfaceError.value || showStaleBanner.value);

/** Block pointer/keyboard into the slot while a cover (error or block busy) is up. */
const slotInert = computed(
  () => showSurfaceError.value || (busy.value && mode.value !== 'stale'),
);

const surfaceCopy = computed(() =>
  resolveViewSurfaceErrorCopy(props.error, {
    fallbackTitle: props.errorFallbackTitle,
  }),
);

const staleBannerCopy = computed(() => resolveViewSurfaceStaleBannerCopy());

let recoveryRetryTimer: ReturnType<typeof setTimeout> | null = null;
let autoRetryResetTimer: ReturnType<typeof setTimeout> | null = null;
/** One auto soft-retry per failure episode (avoids flapping storms). */
let autoRetriedForCurrentError = false;

function clearRecoveryRetryTimer() {
  if (recoveryRetryTimer) {
    clearTimeout(recoveryRetryTimer);
    recoveryRetryTimer = null;
  }
}

function clearAutoRetryResetTimer() {
  if (autoRetryResetTimer) {
    clearTimeout(autoRetryResetTimer);
    autoRetryResetTimer = null;
  }
}

watch(
  () => props.error,
  (next) => {
    if (next != null) {
      clearAutoRetryResetTimer();
      return;
    }
    clearRecoveryRetryTimer();
    clearAutoRetryResetTimer();
    autoRetryResetTimer = setTimeout(() => {
      autoRetryResetTimer = null;
      if (props.error == null) autoRetriedForCurrentError = false;
    }, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 100);
  },
);

// Mounted Action surfaces stay on their error/stale UI after reconnect — Asset
// ChoyViewLoadError already soft-retries; mirror that here via the same generation.
watch(
  () => connectivity.recoveryGeneration,
  (gen, prev) => {
    if (gen <= 0 || (prev != null && gen <= prev)) return;
    if (connectivity.status !== 'reachable') return;
    if (!showRetryableFailure.value || autoRetriedForCurrentError) return;
    clearRecoveryRetryTimer();
    recoveryRetryTimer = setTimeout(() => {
      recoveryRetryTimer = null;
      if (connectivity.status !== 'reachable') return;
      if (!showRetryableFailure.value || autoRetriedForCurrentError) return;
      autoRetriedForCurrentError = true;
      emit('retry');
    }, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
  },
);

onBeforeUnmount(() => {
  clearRecoveryRetryTimer();
  clearAutoRetryResetTimer();
});
</script>
