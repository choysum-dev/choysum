<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Skeleton data-anchor="choy.skeleton" v-bind="$attrs" :class="props.class" />
</template>

<script setup lang="ts">
import Skeleton from '../vendor/ui/skeleton/Skeleton.vue';
import type { ClassValue } from '../../lib/utils';

/**
 * Public L1 skeleton for page/list loading. Domains must not import vendor Skeleton.
 */
defineOptions({ name: 'ChoySkeleton', inheritAttrs: false });

const props = defineProps<{
  class?: ClassValue;
}>();
</script>
