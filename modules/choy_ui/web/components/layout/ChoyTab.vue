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
/** False when register was refused (duplicate value); skip unregister/patch. */
const ownsRegistration = ref(false);

function resolveLabel(): string {
  if (props.label) {
    return props.label;
  }
  return props.value;
}

onMounted(() => {
  registeredValue.value = props.value;
  ownsRegistration.value =
    ctx?.register({
      value: props.value,
      label: resolveLabel(),
      disabled: props.disabled,
    }) ?? true;
});

watch(
  () => [props.value, props.label, props.disabled] as const,
  ([value], [oldValue]) => {
    if (!ownsRegistration.value) {
      return;
    }
    if (oldValue !== value) {
      // Rename in place so the tab keeps its slot; skip if the new value is taken.
      const ok = ctx
        ? ctx.update(registeredValue.value, {
            value,
            label: resolveLabel(),
            disabled: props.disabled,
          })
        : true;
      if (ok) {
        registeredValue.value = value;
      }
      return;
    }
    ctx?.update(registeredValue.value, {
      label: resolveLabel(),
      disabled: props.disabled,
    });
  },
);

onBeforeUnmount(() => {
  if (ownsRegistration.value) {
    ctx?.unregister(registeredValue.value);
  }
});
</script>

<template>
  <TabsContent
    v-if="ownsRegistration"
    data-anchor="choy.tab"
    :value="registeredValue"
    :class="props.class"
  >
    <slot />
  </TabsContent>
</template>
