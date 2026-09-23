<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<script setup lang="ts">
import { inject, onBeforeUnmount, onMounted, ref, watch } from 'vue';
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
/** Value currently registered with the parent (may lag props.value briefly). */
const registeredValue = ref(props.value);

function resolveLabel(): string {
  if (props.label) {
    return props.label;
  }
  return props.value;
}

onMounted(() => {
  registeredValue.value = props.value;
  ctx?.register({
    value: props.value,
    label: resolveLabel(),
    disabled: props.disabled,
  });
});

watch(
  () => [props.value, props.label, props.disabled] as const,
  ([value], [oldValue]) => {
    if (oldValue !== value) {
      ctx?.unregister(registeredValue.value);
      registeredValue.value = value;
      ctx?.register({
        value,
        label: resolveLabel(),
        disabled: props.disabled,
      });
      return;
    }
    ctx?.update(value, {
      label: resolveLabel(),
      disabled: props.disabled,
    });
  },
);

onBeforeUnmount(() => {
  ctx?.unregister(registeredValue.value);
});
</script>

<template>
  <TabsContent data-anchor="choy.tab" :value="value" :class="props.class">
    <slot />
  </TabsContent>
</template>
