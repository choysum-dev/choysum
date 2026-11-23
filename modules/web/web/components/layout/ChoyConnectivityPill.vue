<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Header chrome: same ghost icon button as command palette / notification bell.
    Warning is carried by icon color only — no chip border/fill (Odoo systray-like).
  -->
  <Popover v-if="store.isUnreachable" v-model:open="open">
    <PopoverTrigger as-child>
      <ChoyButton
        type="button"
        variant="ghost"
        size="icon"
        class="choy-connectivity-pill text-warning hover:bg-accent hover:text-warning"
        data-testid="choy-connectivity-pill"
        :aria-label="_t('Connection lost')"
        :title="_t('Connection lost')"
      >
        <Link2Off class="size-4 shrink-0 text-warning" aria-hidden="true" />
      </ChoyButton>
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
import { Link2Off } from 'lucide-vue-next';
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
