<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->
<template>
  <div class="fe-stub-opage" data-testid="fe-stub-opage" :aria-busy="loading ? 'true' : undefined">
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  loading?: boolean;
  title?: string;
}>();
</script>
