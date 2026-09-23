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

watch(
  tabs,
  (list) => {
    if (!list.length) {
      return;
    }
    if (modelValue.value === undefined || modelValue.value === '') {
      modelValue.value = props.defaultValue ?? list[0].value;
    }
  },
  { deep: true },
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
