<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->
<template>
  <div class="auth-coverage-probe" data-testid="auth-coverage-probe">{{ label }}</div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

const props = defineProps<{ title?: string }>();
const label = computed(() => String(props.title || 'auth-probe'));
</script>
