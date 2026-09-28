<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <slot />
</template>

<script setup lang="ts">
import { toRef, provide } from 'vue';
import type { ViewMode, ViewContainer } from './ViewScope.vue';
import {
  VIEW_MODE_KEY,
  VIEW_CONTAINER_KEY,
  FIELD_PREFIX_KEY,
} from './viewScopeKeys';

export type { ViewMode, ViewContainer };

defineOptions({ name: 'ChoyViewScope', inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    viewMode?: ViewMode;
    container?: ViewContainer;
    fieldPrefix?: string;
  }>(),
  {
    viewMode: 'edit',
    container: 'Form',
    fieldPrefix: undefined,
  },
);

const modeRef = toRef(props, 'viewMode');
const containerRef = toRef(props, 'container');
const fieldPrefixRef = toRef(props, 'fieldPrefix');

provide(VIEW_MODE_KEY, modeRef);
provide(VIEW_CONTAINER_KEY, containerRef);
provide(FIELD_PREFIX_KEY, fieldPrefixRef);
</script>
