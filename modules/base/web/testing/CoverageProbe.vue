<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->
<template>
  <div class="base-coverage-probe" data-testid="base-coverage-probe">{{ label }}</div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{ title?: string }>();
const label = computed(() => String(props.title || 'base-probe'));
</script>
