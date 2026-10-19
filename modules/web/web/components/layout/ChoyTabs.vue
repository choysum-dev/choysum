<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Tabs
    v-model="modelValue"
    :data-anchor="props.dataAnchor?.trim() || 'choy.tabs'"
    :data-variant="variant"
    :class="cn('choy-tabs', props.class)"
  >
    <TabsList v-if="tabs.length" :class="listClass">
      <TabsTrigger
        v-for="tab in tabs"
        :key="tab.value"
        :value="tab.value"
        :disabled="tab.disabled"
        :class="triggerClass"
      >
        {{ tab.label }}
      </TabsTrigger>
    </TabsList>
    <slot />
  </Tabs>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue';
import Tabs from '../vendor/ui/tabs/Tabs.vue';
import TabsList from '../vendor/ui/tabs/TabsList.vue';
import TabsTrigger from '../vendor/ui/tabs/TabsTrigger.vue';
import { cn, type ClassValue } from '../../lib/utils';
import {
  ChoyTabsContextKey,
  createChoyTabsContext,
  type ChoyTabRegistration
} from './choyTabsContext';
import { nextChoyTabSelection } from './choyTabsSelection';

export type ChoyTabsVariant = 'line' | 'pill';

/**
 * Public tabs host. Children are ChoyTab (not TabPane).
 * Default variant is line (left-aligned underline) for form notebooks;
 * pass variant="pill" for compact muted toolbars (kit default look).
 * Line styles live in scoped CSS so they win over kit utilities (cn does not tw-merge).
 * `dataAnchor` overrides the default L1 anchor for IMD extension points
 * (e.g. partner.detail.tab-panels).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    defaultValue?: string;
    /** Public IMD extension anchor; defaults to choy.tabs. */
    dataAnchor?: string;
    variant?: ChoyTabsVariant;
  }>(),
  {
    variant: 'line',
  },
);

const modelValue = defineModel<string>();
const tabs = ref<ChoyTabRegistration[]>([]);
const ctx = createChoyTabsContext(tabs, modelValue);
/** True after initial mount; children have already registered by then. */
const settled = ref(false);

const variant = computed(() => props.variant);

/** Marker class for tests; real line chrome is scoped CSS below. */
const listClass = computed(() => (variant.value === 'pill' ? undefined : 'choy-tabs__list--line'));
const triggerClass = computed(() => (variant.value === 'pill' ? undefined : 'choy-tabs__trigger--line'));

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

<style scoped>
/*
 * Line notebook. Kit TabsList/Trigger keep utility classes; cn() concatenates
 * without tw-merge, so product rules must outrank h-control / bg-muted / active pill.
 */
.choy-tabs[data-variant='line'] :deep([data-slot='tabs-list'].choy-tabs__list--line) {
  display: flex;
  height: auto;
  width: 100%;
  align-items: stretch;
  justify-content: flex-start;
  gap: 0;
  padding: 0;
  border-radius: 0;
  border-bottom: 1px solid var(--color-border);
  background-color: transparent;
  color: var(--color-muted-foreground);
}

.choy-tabs[data-variant='line'] :deep([data-slot='tabs-trigger'].choy-tabs__trigger--line) {
  height: auto;
  min-height: var(--choy-control-height, 2rem);
  margin-bottom: -1px;
  border: 0;
  border-bottom: 2px solid transparent;
  border-radius: 0;
  background-color: transparent;
  padding: 0.375rem 0.875rem;
  box-shadow: none;
  font-size: 0.875rem;
  font-weight: 500;
  line-height: 1.25rem;
  color: var(--color-muted-foreground);
}

.choy-tabs[data-variant='line'] :deep([data-slot='tabs-trigger'].choy-tabs__trigger--line:hover) {
  color: var(--color-foreground);
  background-color: transparent;
}

.choy-tabs[data-variant='line']
  :deep([data-slot='tabs-trigger'].choy-tabs__trigger--line[data-state='active']) {
  border-bottom-color: var(--color-primary);
  background-color: transparent;
  box-shadow: none;
  font-weight: 600;
  color: var(--color-foreground);
}

.choy-tabs[data-variant='line'] :deep([data-slot='tabs-content']) {
  margin-top: 0;
  padding-top: 1rem;
}
</style>
