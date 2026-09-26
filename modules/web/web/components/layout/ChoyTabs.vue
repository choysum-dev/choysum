<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Tabs
    v-model="modelValue"
    :data-anchor="props.dataAnchor?.trim() || 'choy.tabs'"
    :class="props.class"
  >
    <TabsList v-if="tabs.length">
      <TabsTrigger
        v-for="tab in tabs"
        :key="tab.value"
        :value="tab.value"
        :disabled="tab.disabled"
      >
        {{ tab.label }}
      </TabsTrigger>
    </TabsList>
    <slot />
  </Tabs>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, provide, ref, watch } from 'vue';
import Tabs from '../vendor/ui/tabs/Tabs.vue';
import TabsList from '../vendor/ui/tabs/TabsList.vue';
import TabsTrigger from '../vendor/ui/tabs/TabsTrigger.vue';
import type { ClassValue } from '../../lib/utils';
import {
  ChoyTabsContextKey,
  createChoyTabsContext,
  type ChoyTabRegistration,
} from './choyTabsContext';
import { nextChoyTabSelection } from './choyTabsSelection';

/**
 * Public tabs host. Children are ChoyTab (not TabPane).
 * `dataAnchor` overrides the default L1 anchor for IMD extension points
 * (e.g. partner.detail.tab-panels).
 */
const props = defineProps<{
  class?: ClassValue;
  defaultValue?: string;
  /** Public IMD extension anchor; defaults to choy.tabs. */
  dataAnchor?: string;
}>();

const modelValue = defineModel<string>();
const tabs = ref<ChoyTabRegistration[]>([]);
const ctx = createChoyTabsContext(tabs);
/** True after initial mount; children have already registered by then. */
const settled = ref(false);

function reconcileSelection(list: ChoyTabRegistration[]): void {
  const next = nextChoyTabSelection(list, modelValue.value, props.defaultValue, settled.value);
  if (next !== undefined) {
    modelValue.value = next;
  }
}

provide(ChoyTabsContextKey, ctx);

watch(tabs, (list) => reconcileSelection(list));

// Late-arriving or parent-swapped defaultValue must also reconcile.
watch(
  () => props.defaultValue,
  () => reconcileSelection(tabs.value),
);

onMounted(() => {
  settled.value = true;
  reconcileSelection(tabs.value);
});

onBeforeUnmount(() => {
  tabs.value = [];
});
</script>
