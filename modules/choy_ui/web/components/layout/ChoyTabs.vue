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
  type ChoyTabRegistration,
  type ChoyTabsContext,
} from './choyTabsContext';

/**
 * Public tabs host. Children are ChoyTab (not TabPane).
 */
const props = defineProps<{
  class?: ClassValue;
  defaultValue?: string;
}>();

const modelValue = defineModel<string>();
const tabs = ref<ChoyTabRegistration[]>([]);

function firstEnabledValue(list: ChoyTabRegistration[]): string {
  const enabled = list.find((item) => !item.disabled);
  return (enabled ?? list[0])?.value ?? '';
}

/**
 * Picks a valid selection: prefer defaultValue when it names an enabled tab,
 * otherwise the first enabled registration.
 */
function pickSelection(list: ChoyTabRegistration[]): string {
  const def = props.defaultValue;
  if (def && list.some((item) => item.value === def && !item.disabled)) {
    return def;
  }
  return firstEnabledValue(list);
}

function reconcileSelection(list: ChoyTabRegistration[]): void {
  if (!list.length) {
    return;
  }
  const current = modelValue.value;
  const stillValid =
    !!current && list.some((item) => item.value === current && !item.disabled);
  if (!stillValid) {
    modelValue.value = pickSelection(list);
  }
}

const ctx: ChoyTabsContext = {
  tabs,
  register(tab) {
    if (tabs.value.some((item) => item.value === tab.value)) {
      return;
    }
    tabs.value = [...tabs.value, tab];
  },
  unregister(value) {
    tabs.value = tabs.value.filter((item) => item.value !== value);
  },
  update(value, patch) {
    tabs.value = tabs.value.map((item) => (item.value === value ? { ...item, ...patch } : item));
  },
};

provide(ChoyTabsContextKey, ctx);

watch(tabs, (list) => reconcileSelection(list), { deep: true });

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
