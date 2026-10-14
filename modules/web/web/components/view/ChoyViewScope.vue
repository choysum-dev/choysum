<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <slot />
</template>

<script setup lang="ts">
import { computed, watchEffect } from 'vue';
import { getViewKind } from './viewKinds';
import { useProvideViewScope, type ViewMode } from './viewScopeKeys';
import type { ViewKindId } from './viewKinds';
import { registerBuiltinViewKinds } from './viewKinds';

export type { ViewMode };
export type { ViewKindId };
/** @deprecated Use ViewKindId */
export type ViewContainer = ViewKindId;

defineOptions({ name: 'ChoyViewScope', inheritAttrs: false });

registerBuiltinViewKinds();

const props = defineProps<{
  viewMode?: ViewMode;
  /** Registered view kind id (Form, List, Kanban, …). */
  kind?: ViewKindId;
  /**
   * @deprecated Use `kind`. Accepted only to surface a clear warning when
   * call sites still pass the removed prop (inheritAttrs is false).
   */
  container?: ViewKindId;
  fieldPrefix?: string;
}>();

watchEffect(() => {
  if (props.container !== undefined && props.kind === undefined) {
    console.warn('[ChoyViewScope] prop "container" was removed; use "kind" instead');
  }
});

const kind = computed<ViewKindId>(() => {
  if (props.kind !== undefined) return props.kind;
  if (props.container !== undefined) return props.container;
  return 'Form';
});

/** Prefer explicit viewMode; otherwise the registered kind defaultMode. */
const mode = computed<ViewMode>(
  () => props.viewMode ?? getViewKind(kind.value).defaultMode ?? 'display',
);

useProvideViewScope({
  kind,
  mode,
  // Absent prop resets to null so relation-picker dialogs do not inherit an enclosing prefix.
  fieldPrefix: computed(() => props.fieldPrefix ?? null),
});
</script>
