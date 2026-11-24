<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div
    v-if="error"
    class="choy-view-load-error flex min-h-0 flex-1 flex-col items-center justify-center gap-3 px-6 py-12 text-center"
    data-testid="choy-view-load-error"
  >
    <p class="text-base font-medium text-foreground">{{ title }}</p>
    <p class="max-w-md text-sm text-muted-foreground">{{ detail }}</p>
    <div class="flex flex-wrap items-center justify-center gap-2">
      <ChoyButton type="button" data-testid="choy-view-load-error-retry" @click="onRetry">
        {{ _t('Retry loading') }}
      </ChoyButton>
      <ChoyButton type="button" variant="outline" data-testid="choy-view-load-error-reload" @click="onReload">
        {{ _t('Reload page') }}
      </ChoyButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, watch } from 'vue';
import { useRouter } from 'vue-router';
import ChoyButton from '../button/ChoyButton.vue';
import { createTranslate } from '@/web/web/i18n';
import { useConnectivityStore } from '../../stores/connectivityStore';
import {
  clearViewLoadError,
  VIEW_LOAD_RETRY_AFTER_RECOVERY_MS,
  viewLoadError as viewLoadErrorRef,
} from '../../router/viewLoadError';

defineOptions({ name: 'ChoyViewLoadError' });

const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyViewLoadError' });
const router = useRouter();
const connectivity = useConnectivityStore();
const error = viewLoadErrorRef;

const title = computed(() =>
  connectivity.isUnreachable ? _t('Connection lost') : _t('Failed to load this view')
);
const detail = computed(() =>
  connectivity.isUnreachable
    ? _t('Cannot reach the server. Check your network, then retry.')
    : _t('The page resources could not be loaded. Retry or reload the page.')
);

async function onRetry(): Promise<void> {
  const fullPath = error.value?.fullPath || router.currentRoute.value.fullPath;
  clearViewLoadError();
  // Same-path replace is a duplicated navigation and will not re-import a failed
  // lazy chunk — soft history reload re-enters the route and retries the import.
  if (String(fullPath) === String(router.currentRoute.value.fullPath || '')) {
    router.go(0);
    return;
  }
  try {
    await router.replace(fullPath);
  } catch {
    // onError may re-set viewLoadError
  }
}

function onReload(): void {
  if (typeof window !== 'undefined') {
    window.location.reload();
  }
}

let recoveryRetryTimer: ReturnType<typeof setTimeout> | null = null;
let autoRetryResetTimer: ReturnType<typeof setTimeout> | null = null;
/** One auto soft-retry per error path until a clear sticks (avoids flapping storms). */
let autoRetriedForCurrentError = false;
/** User started navigating to a different route while the failed view error remains. */
let navigatingAwayFromError = false;

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

function currentFullPath(): string {
  return String(router.currentRoute.value.fullPath || '');
}

watch(error, (next) => {
  if (next) {
    // Soft-retry clears then may re-set the same path — keep the one-shot flag.
    clearAutoRetryResetTimer();
    return;
  }
  clearRecoveryRetryTimer();
  navigatingAwayFromError = false;
  // Only reset the one-shot after a clear that sticks (failed soft-retry re-sets sooner).
  clearAutoRetryResetTimer();
  autoRetryResetTimer = setTimeout(() => {
    autoRetryResetTimer = null;
    if (!error.value) autoRetriedForCurrentError = false;
  }, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS + 100);
});

const stopLeaveGuard = router.beforeEach((to) => {
  if (!error.value) return;
  if (String(to.fullPath) === String(error.value.fullPath)) {
    navigatingAwayFromError = false;
    return;
  }
  navigatingAwayFromError = true;
  clearRecoveryRetryTimer();
});

// Release the leave flag once that navigation settles so a cancelled / ignored
// leave does not permanently disable recovery soft-retry.
const stopAfterEach = router.afterEach(() => {
  if (!navigatingAwayFromError) return;
  navigatingAwayFromError = false;
});

// Full-page load error has no editable draft — soft-retry after reconnect.
watch(
  () => connectivity.recoveryGeneration,
  (gen, prev) => {
    if (gen <= 0 || (prev != null && gen <= prev)) return;
    if (!error.value || connectivity.status !== 'reachable') return;
    if (autoRetriedForCurrentError || navigatingAwayFromError) return;
    const errPath = error.value.fullPath;
    // Chunk failures can leave currentRoute on the previous page while error.fullPath
    // is the failed destination — capture current and require it unchanged at fire.
    const routeAtSchedule = currentFullPath();
    clearRecoveryRetryTimer();
    recoveryRetryTimer = setTimeout(() => {
      recoveryRetryTimer = null;
      if (!error.value || connectivity.status !== 'reachable') return;
      if (autoRetriedForCurrentError || navigatingAwayFromError) return;
      if (String(error.value.fullPath) !== String(errPath)) return;
      if (currentFullPath() !== routeAtSchedule) return;
      autoRetriedForCurrentError = true;
      void onRetry();
    }, VIEW_LOAD_RETRY_AFTER_RECOVERY_MS);
  },
);

onBeforeUnmount(() => {
  clearRecoveryRetryTimer();
  clearAutoRetryResetTimer();
  stopLeaveGuard();
  stopAfterEach();
});
</script>
