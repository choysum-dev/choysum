<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <Spinner
    data-testid="choy-spinner"
    :class="['choy-spinner', props.class]"
    :aria-label="label || 'Loading'"
  />
</template>

<script setup lang="ts">
import type { ClassValue } from '../../lib/utils';
import Spinner from '../vendor/ui/spinner/Spinner.vue';

/**
 * Product loading spinner over vendor Spinner.
 */
const props = defineProps<{
  class?: ClassValue;
  label?: string;
}>();
</script>
