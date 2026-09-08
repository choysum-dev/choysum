<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: LGPL-3.0-or-later
-->
<template>
  <div class="fe-stub-child-view" data-testid="fe-stub-child-view">
    <slot />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  initialValues?: Record<string, unknown>;
  moduleStore?: unknown;
}>();
</script>
