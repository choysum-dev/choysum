<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Popover v-if="store.isUnreachable" v-model:open="open">
    <PopoverTrigger as-child>
      <button
        type="button"
        class="choy-connectivity-pill inline-flex h-8 items-center gap-1.5 rounded-md border border-amber-500/40 bg-amber-500/15 px-2.5 text-xs font-medium text-amber-900 dark:text-amber-100"
        data-testid="choy-connectivity-pill"
        :aria-label="_t('Connection lost')"
      >
        <CloudOff class="size-3.5 shrink-0" aria-hidden="true" />
        <span class="max-w-[10rem] truncate">{{ _t('Connection lost') }}</span>
      </button>
    </PopoverTrigger>
    <PopoverContent align="end" class="w-72 space-y-3 p-3" data-testid="choy-connectivity-pill-panel">
      <p class="text-sm text-foreground">
        {{ _t('Cannot reach the server. Check your network, then retry.') }}
      </p>
      <ChoyButton
        type="button"
        size="sm"
        class="w-full"
        :disabled="store.probing"
        data-testid="choy-connectivity-pill-retry"
        @click="onRetry"
      >
        {{ store.probing ? _t('Checking…') : _t('Retry') }}
      </ChoyButton>
    </PopoverContent>
  </Popover>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { CloudOff } from 'lucide-vue-next';
import { Popover, PopoverContent, PopoverTrigger } from '../vendor/ui/popover';
import ChoyButton from './ChoyButton.vue';
import { createTranslate } from '@/web/web/i18n';
import { useConnectivityStore } from '../../stores/connectivityStore';

defineOptions({ name: 'ChoyConnectivityPill' });

const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyConnectivityPill' });
const store = useConnectivityStore();
const open = ref(false);

async function onRetry(): Promise<void> {
  const ok = await store.probeConnectivity();
  if (ok) open.value = false;
}
</script>
