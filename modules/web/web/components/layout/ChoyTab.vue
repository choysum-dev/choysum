<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div v-if="!ctx" data-anchor="choy.tab" :class="props.class">
    <slot />
  </div>
  <TabsContent
    v-else-if="ownsRegistration"
    data-anchor="choy.tab"
    :value="registeredValue"
    :force-mount="forceMount"
    :class="contentClass"
  >
    <!-- forceMount keeps inactive panes mounted so Form returnFields stay registered. -->
    <slot v-if="isActive || forceMount" />
  </TabsContent>
</template>

<script setup lang="ts">
import { computed, inject, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import TabsContent from '../vendor/ui/tabs/TabsContent.vue';
import { cn, type ClassValue } from '../../lib/utils';
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
    /**
     * Keep pane content mounted while inactive (hidden). Use when sibling tabs
     * share Form field registration (e.g. O2M table + kanban for the same prop).
     */
    forceMount?: boolean;
  }>(),
  {
    label: '',
    disabled: false,
    forceMount: false,
  },
);

const ctx = inject(ChoyTabsContextKey, null);
/** Value currently registered with the parent (may lag props.value briefly). */
const registeredValue = ref(props.value);
const isActive = computed(() => !!ctx && ctx.activeValue.value === registeredValue.value);
/** False when register was refused (duplicate value); skip unregister/patch. */
const ownsRegistration = ref(false);
const contentClass = computed(() =>
  cn(props.class, props.forceMount && !isActive.value ? 'hidden' : undefined)
);

function resolveLabel(): string {
  if (props.label) {
    return props.label;
  }
  return props.value;
}

function tryRegister(value: string): boolean {
  if (!ctx) {
    return true;
  }
  return ctx.register({
    value,
    label: resolveLabel(),
    disabled: props.disabled,
  });
}

onMounted(() => {
  registeredValue.value = props.value;
  ownsRegistration.value = tryRegister(props.value);
});

// forceMount panes start as `hidden`; notify layout consumers when they become visible
// so embedded virtual tables/boards remasure scrollports (ResizeObserver alone can miss).
watch(isActive, (active, wasActive) => {
  if (!props.forceMount || !active || wasActive) return;
  void Promise.resolve().then(() => {
    if (typeof window === 'undefined') return;
    window.dispatchEvent(new Event('resize'));
  });
});

// Retry when the shared registry changes: a conflicting tab may release the
// target value (refused register or collided rename) without props.value changing.
watch(
  () => ctx?.tabs.value.some((item) => item.value === props.value) ?? false,
  (taken) => {
    if (ownsRegistration.value) {
      // A previous rename may have collided; retry once the value frees up.
      if (!taken && registeredValue.value !== props.value) {
        const renamed = ctx
          ? ctx.update(registeredValue.value, {
              value: props.value,
              label: resolveLabel(),
              disabled: props.disabled,
            })
          : true;
        if (renamed) {
          registeredValue.value = props.value;
        }
      }
      return;
    }
    if (taken) {
      return;
    }
    if (tryRegister(props.value)) {
      ownsRegistration.value = true;
      registeredValue.value = props.value;
    }
  },
);

watch(
  () => [props.value, props.label, props.disabled] as const,
  ([value], [oldValue]) => {
    if (!ownsRegistration.value) {
      // Refused duplicate must not hide forever: retry when value becomes free.
      if (oldValue !== value && tryRegister(value)) {
        ownsRegistration.value = true;
        registeredValue.value = value;
      }
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
      } else {
        // Rename lost the race for `value`; keep label/disabled on the owned value.
        ctx?.update(registeredValue.value, {
          label: props.label || registeredValue.value,
          disabled: props.disabled,
        });
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
