<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <OVirtualField v-if="storeMode" v-bind="(storeBind as any)" />
  <!-- Chrome: virtual field registers value only; no DOM. -->
</template>

<script setup lang="ts">
import { computed, useAttrs } from 'vue';
import type { WebModelStore } from '@/web/web/stores/modelStore';
import { isChoyStoreFieldBinding } from '@/web/web/composables/choyStoreMode';
import OVirtualField from './OVirtualField.vue';

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
const storeMode = computed(() => isChoyStoreFieldBinding(props));
const storeBind = computed(() => ({ ...attrs, ...props }) as any);

defineModel<unknown>({ default: null });
</script>
