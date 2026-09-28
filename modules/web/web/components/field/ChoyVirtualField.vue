<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <VirtualField v-if="storeMode" v-bind="(storeBind as any)" />
  <!-- Chrome: virtual field registers value only; no DOM. -->
</template>

<script setup lang="ts">
import { useAttrs } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { useChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import VirtualField from './VirtualField.vue';

defineOptions({ name: 'ChoyVirtualField', inheritAttrs: false });

/**
 * Virtual field: store+prop hosts OVirtualField; otherwise holds v-model only.
 */
const props = defineProps<{
  store?: WebModelStore<any>;
  prop?: string;
  binding?: unknown;
}>();

const attrs = useAttrs();
const { storeMode, storeBind } = useChoyStoreFieldBinding(props as any, attrs as Record<string, unknown>);

defineModel<unknown>({ default: null });
</script>
