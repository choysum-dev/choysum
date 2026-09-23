<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { onBeforeUnmount, provide, ref, watch } from 'vue';
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
 */
const props = defineProps<{
  class?: ClassValue;
  defaultValue?: string;
}>();

const modelValue = defineModel<string>();
const tabs = ref<ChoyTabRegistration[]>([]);
const ctx = createChoyTabsContext(tabs);

function reconcileSelection(list: ChoyTabRegistration[]): void {
  const next = nextChoyTabSelection(list, modelValue.value, props.defaultValue);
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

onBeforeUnmount(() => {
  tabs.value = [];
});
</script>

<template>
  <Tabs
    v-model="modelValue"
    data-anchor="choy.tabs"
    :class="props.class"
    :default-value="defaultValue"
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
