<!--
SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
SPDX-License-Identifier: Apache-2.0
-->

<template>
  <div data-testid="hello">Hello {{ name }}</div>
</template>

<script setup lang="ts">
defineProps<{ name: string }>();
</script>
