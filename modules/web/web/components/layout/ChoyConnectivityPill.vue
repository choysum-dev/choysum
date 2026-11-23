<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!--
    Session connectivity chrome (sidebar footer when expanded). Ghost icon
    button; click probes immediately. Warning is icon color only — no panel.
  -->
  <ChoyButton
    v-if="store.isUnreachable"
    type="button"
    variant="ghost"
    size="icon"
    class="choy-connectivity-pill text-warning hover:bg-accent hover:text-warning"
    data-testid="choy-connectivity-pill"
    :aria-label="ariaLabel"
    :title="ariaLabel"
    :disabled="store.probing"
    @click="onRetry"
  >
    <Link2Off class="size-4 shrink-0" aria-hidden="true" />
  </ChoyButton>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { Link2Off } from 'lucide-vue-next';
import ChoyButton from './ChoyButton.vue';
import { createTranslate } from '@/web/web/i18n';
import { useConnectivityStore } from '../../stores/connectivityStore';

defineOptions({ name: 'ChoyConnectivityPill' });

const { _t } = createTranslate('web', { scope: 'web/components/layout/ChoyConnectivityPill' });
const store = useConnectivityStore();

const ariaLabel = computed(() =>
  store.probing
    ? _t('Checking…')
    : `${_t('Connection lost')}. ${_t('Retry connection')}`,
);

async function onRetry(): Promise<void> {
  await store.probeConnectivity();
}
</script>
