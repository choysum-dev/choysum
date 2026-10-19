<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Badge data-anchor="choy.badge" v-bind="$attrs" :class="props.class" :variant="props.variant">
    <slot />
  </Badge>
</template>

<script setup lang="ts">
import Badge from '../vendor/ui/badge/Badge.vue';
import type { ClassValue } from '../../lib/utils';

/**
 * Public L1 badge for counts / status chips. Domains must not import vendor Badge.
 */
defineOptions({ name: 'ChoyBadge', inheritAttrs: false });

const props = withDefaults(
  defineProps<{
    class?: ClassValue;
    variant?: 'default' | 'secondary' | 'destructive' | 'outline';
  }>(),
  {
    variant: 'default',
  },
);
</script>
