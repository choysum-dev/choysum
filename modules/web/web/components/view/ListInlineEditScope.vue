<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <slot />
</template>

<script setup lang="ts">
/**
 * Scopes S2 inline-edit form-root + view mode to the list table only.
 * Keeps header search / action-bar fields from entering edit mode or
 * reading/writing the active row draft via list-wide provides.
 */
import { provide, toRef } from 'vue';
import { useProvideViewScope, type ViewMode } from './viewScopeKeys';

defineOptions({ name: 'ListInlineEditScope' });

const props = defineProps<{
  formRoot: {
    draft: unknown;
    getField: (path: string) => unknown;
    setField: (path: string, value: any) => void;
  };
  viewMode: ViewMode;
}>();

useProvideViewScope({
  kind: 'List',
  mode: toRef(props, 'viewMode'),
});

// Stable object identity; getters read live draft state from the composable.
provide('form-root', props.formRoot);
</script>
