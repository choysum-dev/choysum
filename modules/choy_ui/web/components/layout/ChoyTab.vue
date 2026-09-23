<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { inject, onBeforeUnmount, onMounted, watch } from 'vue';
import TabsContent from '../vendor/ui/tabs/TabsContent.vue';
import type { ClassValue } from '../../lib/utils';
import { ChoyTabsContextKey } from './choyTabsContext';

/**
 * Public tab pane. Label comes from the `label` prop (or falls back to value).
 */
const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    value: string;
    label?: string;
    disabled?: boolean;
  }>(),
  {
    label: '',
    disabled: false,
  },
);

const ctx = inject(ChoyTabsContextKey, null);

function resolveLabel(): string {
  if (props.label) {
    return props.label;
  }
  return props.value;
}

onMounted(() => {
  ctx?.register({
    value: props.value,
    label: resolveLabel(),
    disabled: props.disabled,
  });
});

watch(
  () => [props.label, props.disabled] as const,
  () => {
    ctx?.update(props.value, {
      label: resolveLabel(),
      disabled: props.disabled,
    });
  },
);

onBeforeUnmount(() => {
  ctx?.unregister(props.value);
});
</script>

<template>
  <TabsContent data-anchor="choy.tab" :value="value" :class="props.class">
    <slot />
  </TabsContent>
</template>
