<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <slot />
</template>

<script setup lang="ts">
import { toRef } from 'vue';
import { useProvideViewScope, type ViewMode } from './viewScopeKeys';
import type { ViewKindId } from './viewKinds';
import { registerBuiltinViewKinds } from './viewKinds';

export type { ViewMode };
export type { ViewKindId };
/** @deprecated Use ViewKindId */
export type ViewContainer = ViewKindId;

defineOptions({ name: 'ChoyViewScope', inheritAttrs: false });

registerBuiltinViewKinds();

const props = withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    /** Registered view kind id (Form, List, Kanban, …). */
    kind?: ViewKindId;
    fieldPrefix?: string;
  }>(),
  {
    viewMode: 'edit',
    kind: 'Form',
    fieldPrefix: undefined,
  },
);

useProvideViewScope({
  kind: toRef(props, 'kind'),
  mode: toRef(props, 'viewMode'),
  fieldPrefix: toRef(props, 'fieldPrefix'),
});
</script>
