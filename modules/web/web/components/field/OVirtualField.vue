<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <!-- No DOM output: registration-only component. -->
</template>

<script setup lang="ts" generic="T extends BaseModel, P extends FieldPath<T, any>">
import type { BaseModel, FieldPath } from '@/core/rpc';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { watch, onMounted } from 'vue';
import { useField } from '@/web/web/composables/useField';
import type { AggProp } from '@/web/web/composables/useField';

defineOptions({ name: 'OVirtualField' });

type IsAny<T> = 0 extends 1 & T ? true : false;

const props = withDefaults(
  defineProps<{
    store: WebModelStore<T>;
    prop: P | (IsAny<T> extends true ? string : never);
    // Support one or more aggregate declarations.
    agg?: AggProp | AggProp[];
    autoRegister?: boolean; // Register the primary field automatically by default.
    register?: string | string[]; // Extra fields to register in a batch.
    once?: boolean; // If true, register only once and stop watching register.
    debug?: boolean; // Emit debug logs.
  }>(),
  {
    autoRegister: true,
    once: false,
    debug: false,
  }
);

// When autoRegister is enabled, the primary prop is registered automatically.
// Additional paths passed through register are batch-registered for later queries.

// Base binding used only for field registration.
const binding = useField<T, P, any>({
  store: props.store,
  prop: props.prop as unknown as P,
  agg: props.agg,
  autoRegister: props.autoRegister,
});

if (props.debug) {
  // Delay logging until registration has completed.
  queueMicrotask(() => {
    const aggsDesc = Array.isArray(props.agg)
      ? props.agg.map(a => (typeof a === 'string' ? a : `${a.agg}${a.alias ? ':' + a.alias : ''}`)).join(',')
      : props.agg
        ? typeof props.agg === 'string'
          ? props.agg
          : `${props.agg.agg}${props.agg.alias ? ':' + props.agg.alias : ''}`
        : 'none';
    // eslint-disable-next-line no-console
    console.debug('[OVirtualField] registered', { field: props.prop, aggs: aggsDesc, extra: props.register });
  });
}

// Normalize an optional value into an array.
function toArray<T>(v?: T | T[] | null): T[] {
  if (v == null) return [];
  return Array.isArray(v) ? v : [v];
}

// Register extra fields on mount and when the input changes.
function registerExtraFields() {
  const extras = toArray(props.register).filter(Boolean) as string[];
  if (extras.length) binding.registerFields(extras);
}

onMounted(registerExtraFields);
if (!props.once) {
  watch(
    () => props.register,
    () => registerExtraFields(),
    { deep: true }
  );
}
</script>

<style scoped>
/* No styles. */
</style>
