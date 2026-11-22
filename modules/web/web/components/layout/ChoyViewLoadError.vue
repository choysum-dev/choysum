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
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import ChoyButton from './ChoyButton.vue';
import { createTranslate } from '@/web/web/i18n';
import { useConnectivityStore } from '../../stores/connectivityStore';
import { clearViewLoadError, viewLoadError as viewLoadErrorRef } from '../../router/viewLoadError';

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
</script>
